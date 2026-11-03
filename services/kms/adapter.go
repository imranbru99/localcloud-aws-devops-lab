package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type KMSAdapter struct {
	mu      sync.RWMutex
	keys    map[string]*KeyMeta
	aliases map[string]string // aliasName -> keyId
}

type KeyMeta struct {
	KeyId        string `json:"KeyId"`
	Arn          string `json:"Arn"`
	AWSAccountId string `json:"AWSAccountId"`
	Description  string `json:"Description"`
	Enabled      bool   `json:"Enabled"`
	KeyUsage     string `json:"KeyUsage"`
	KeySpec      string `json:"KeySpec"`
	RawKey       []byte `json:"-"`
}

func init() {
	services.Register(&KMSAdapter{
		keys:    make(map[string]*KeyMeta),
		aliases: make(map[string]string),
	})
}

func (a *KMSAdapter) Name() string {
	return "kms"
}

func (a *KMSAdapter) Protocol() services.Protocol {
	return services.ProtocolJSON11
}

func (a *KMSAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateKey":       a.CreateKey,
		"DescribeKey":     a.DescribeKey,
		"ListKeys":        a.ListKeys,
		"CreateAlias":     a.CreateAlias,
		"ListAliases":     a.ListAliases,
		"Encrypt":         a.Encrypt,
		"Decrypt":         a.Decrypt,
		"GenerateDataKey": a.GenerateDataKey,
	}
}

func (a *KMSAdapter) CreateKey(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		Description string `json:"Description"`
		KeyUsage    string `json:"KeyUsage"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	keyId := uuid.New().String()
	arn := fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", ctx.Region, ctx.AccountID, keyId)

	// 256-bit symmetric key
	rawKey := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, rawKey)

	k := &KeyMeta{
		KeyId:        keyId,
		Arn:          arn,
		AWSAccountId: ctx.AccountID,
		Description:  req.Description,
		Enabled:      true,
		KeyUsage:     "ENCRYPT_DECRYPT",
		KeySpec:      "SYMMETRIC_DEFAULT",
		RawKey:       rawKey,
	}
	a.keys[keyId] = k

	return map[string]interface{}{"KeyMetadata": k}, nil
}

func (a *KMSAdapter) DescribeKey(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		KeyId string `json:"KeyId"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	k, ok := a.keys[req.KeyId]
	if !ok {
		// check alias
		if id, has := a.aliases[req.KeyId]; has {
			k = a.keys[id]
		}
	}
	if k == nil {
		return nil, fmt.Errorf("NotFoundException: Key does not exist")
	}

	return map[string]interface{}{"KeyMetadata": k}, nil
}

func (a *KMSAdapter) ListKeys(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var keyList []map[string]string
	for id, k := range a.keys {
		keyList = append(keyList, map[string]string{
			"KeyId":  id,
			"KeyArn": k.Arn,
		})
	}

	return map[string]interface{}{"Keys": keyList, "Truncated": false}, nil
}

func (a *KMSAdapter) CreateAlias(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		AliasName   string `json:"AliasName"`
		TargetKeyId string `json:"TargetKeyId"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	a.aliases[req.AliasName] = req.TargetKeyId
	return map[string]interface{}{}, nil
}

func (a *KMSAdapter) ListAliases(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var aliasList []map[string]string
	for name, id := range a.aliases {
		aliasList = append(aliasList, map[string]string{
			"AliasName":   name,
			"TargetKeyId": id,
			"AliasArn":    fmt.Sprintf("arn:aws:kms:%s:%s:%s", ctx.Region, ctx.AccountID, name),
		})
	}

	return map[string]interface{}{"Aliases": aliasList, "Truncated": false}, nil
}

func (a *KMSAdapter) Encrypt(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		KeyId     string `json:"KeyId"`
		Plaintext string `json:"Plaintext"` // base64
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	k, ok := a.keys[req.KeyId]
	if !ok {
		if id, has := a.aliases[req.KeyId]; has {
			k = a.keys[id]
		}
	}
	if k == nil {
		return nil, fmt.Errorf("NotFoundException")
	}

	plaintextBytes, _ := base64.StdEncoding.DecodeString(req.Plaintext)
	block, _ := aes.NewCipher(k.RawKey)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, plaintextBytes, nil)

	return map[string]interface{}{
		"CiphertextBlob": base64.StdEncoding.EncodeToString(ciphertext),
		"KeyId":          k.Arn,
	}, nil
}

func (a *KMSAdapter) Decrypt(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		CiphertextBlob string `json:"CiphertextBlob"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	blob, err := base64.StdEncoding.DecodeString(req.CiphertextBlob)
	if err != nil || len(blob) < 12 {
		return nil, fmt.Errorf("InvalidCiphertextException")
	}

	// Try each key
	for _, k := range a.keys {
		block, err := aes.NewCipher(k.RawKey)
		if err != nil {
			continue
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			continue
		}
		nonceSize := gcm.NonceSize()
		if len(blob) < nonceSize {
			continue
		}
		nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
		plain, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err == nil {
			return map[string]interface{}{
				"KeyId":     k.Arn,
				"Plaintext": base64.StdEncoding.EncodeToString(plain),
			}, nil
		}
	}

	return nil, fmt.Errorf("InvalidCiphertextException: Decryption failed")
}

func (a *KMSAdapter) GenerateDataKey(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		KeyId string `json:"KeyId"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	k, ok := a.keys[req.KeyId]
	if !ok {
		return nil, fmt.Errorf("NotFoundException")
	}

	dataKey := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, dataKey)

	block, _ := aes.NewCipher(k.RawKey)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, dataKey, nil)

	return map[string]interface{}{
		"KeyId":          k.Arn,
		"Plaintext":      base64.StdEncoding.EncodeToString(dataKey),
		"CiphertextBlob": base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func (a *KMSAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Key", ARNFormat: "arn:aws:kms:{region}:{account}:key/{keyid}", Tags: true, Columns: []string{"KeyId", "Arn", "Description", "Enabled"}},
	}
}

func (a *KMSAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "kms:CreateKey", ResourceARNs: []string{"*"}},
		{Name: "kms:Encrypt", ResourceARNs: []string{"arn:aws:kms:*:*:key/*"}},
		{Name: "kms:Decrypt", ResourceARNs: []string{"arn:aws:kms:*:*:key/*"}},
	}
}

func (a *KMSAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *KMSAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "CreateKey", Source: "kms.amazonaws.com"},
	}
}

func (a *KMSAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *KMSAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *KMSAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.keys = make(map[string]*KeyMeta)
	a.aliases = make(map[string]string)
	return nil
}

func (a *KMSAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *KMSAdapter) Restore(data map[string]interface{}) error {
	return nil
}
