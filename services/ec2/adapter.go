package ec2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type EC2Adapter struct {
	mu             sync.RWMutex
	instances      map[string]*InstanceMeta
	keyPairs       map[string]*KeyPairMeta
	securityGroups map[string]*SecurityGroupMeta
	vpcs           map[string]*VpcMeta
	subnets        map[string]*SubnetMeta
}

type InstanceMeta struct {
	InstanceId   string
	ImageId      string
	InstanceType string
	State        string
	PrivateIp    string
	PublicIp     string
	LaunchTime   time.Time
	VpcId        string
	SubnetId     string
}

type KeyPairMeta struct {
	KeyName        string
	KeyFingerprint string
	KeyMaterial    string
}

type SecurityGroupMeta struct {
	GroupId     string
	GroupName   string
	Description string
	VpcId       string
}

type VpcMeta struct {
	VpcId     string
	CidrBlock string
	State     string
}

type SubnetMeta struct {
	SubnetId  string
	VpcId     string
	CidrBlock string
	State     string
}

func init() {
	services.Register(&EC2Adapter{
		instances:      make(map[string]*InstanceMeta),
		keyPairs:       make(map[string]*KeyPairMeta),
		securityGroups: make(map[string]*SecurityGroupMeta),
		vpcs:           make(map[string]*VpcMeta),
		subnets:        make(map[string]*SubnetMeta),
	})
}

func (a *EC2Adapter) Name() string {
	return "ec2"
}

func (a *EC2Adapter) Protocol() services.Protocol {
	return services.ProtocolEC2
}

func (a *EC2Adapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"DescribeInstances":           a.DescribeInstances,
		"RunInstances":                 a.RunInstances,
		"TerminateInstances":          a.TerminateInstances,
		"StopInstances":               a.StopInstances,
		"StartInstances":              a.StartInstances,
		"DescribeKeyPairs":            a.DescribeKeyPairs,
		"CreateKeyPair":               a.CreateKeyPair,
		"DescribeSecurityGroups":      a.DescribeSecurityGroups,
		"CreateSecurityGroup":         a.CreateSecurityGroup,
		"AuthorizeSecurityGroupIngress": a.AuthorizeSecurityGroupIngress,
		"DescribeVpcs":                a.DescribeVpcs,
		"CreateVpc":                   a.CreateVpc,
		"DescribeSubnets":             a.DescribeSubnets,
		"CreateSubnet":                a.CreateSubnet,
	}
}

func (a *EC2Adapter) DescribeInstances(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var itemsXML string
	for _, inst := range a.instances {
		itemsXML += fmt.Sprintf(`        <item>
          <instanceId>%s</instanceId>
          <imageId>%s</imageId>
          <instanceState>
            <code>16</code>
            <name>%s</name>
          </instanceState>
          <privateDnsName>ip-%s.ec2.internal</privateDnsName>
          <dnsName>ec2-%s.compute-1.amazonaws.com</dnsName>
          <instanceType>%s</instanceType>
          <launchTime>%s</launchTime>
          <ipAddress>%s</ipAddress>
          <privateIpAddress>%s</privateIpAddress>
          <vpcId>%s</vpcId>
          <subnetId>%s</subnetId>
        </item>
`, inst.InstanceId, inst.ImageId, inst.State, inst.PrivateIp, inst.PublicIp, inst.InstanceType, inst.LaunchTime.Format(time.RFC3339), inst.PublicIp, inst.PrivateIp, inst.VpcId, inst.SubnetId)
	}

	xml := fmt.Sprintf(`  <reservationSet>
    <item>
      <reservationId>r-%s</reservationId>
      <ownerId>%s</ownerId>
      <instancesSet>
%s      </instancesSet>
    </item>
  </reservationSet>`, uuid.New().String()[:8], ctx.AccountID, itemsXML)

	return []byte(xml), nil
}

func (a *EC2Adapter) RunInstances(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	imageId := ctx.ParsedParams["ImageId"]
	if imageId == "" {
		imageId = "ami-0c55b159cbfafe1f0"
	}
	instType := ctx.ParsedParams["InstanceType"]
	if instType == "" {
		instType = "t3.micro"
	}

	instId := "i-" + uuid.New().String()[:17]
	inst := &InstanceMeta{
		InstanceId:   instId,
		ImageId:      imageId,
		InstanceType: instType,
		State:        "running",
		PrivateIp:    "10.0.1.10",
		PublicIp:     "54.210.12.34",
		LaunchTime:   time.Now().UTC(),
		VpcId:        "vpc-default",
		SubnetId:     "subnet-default",
	}

	a.instances[instId] = inst

	xml := fmt.Sprintf(`  <instancesSet>
    <item>
      <instanceId>%s</instanceId>
      <imageId>%s</imageId>
      <instanceState>
        <code>16</code>
        <name>running</name>
      </instanceState>
      <instanceType>%s</instanceType>
      <launchTime>%s</launchTime>
    </item>
  </instancesSet>`, inst.InstanceId, inst.ImageId, inst.InstanceType, inst.LaunchTime.Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *EC2Adapter) TerminateInstances(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	instId := ctx.ParsedParams["InstanceId.1"]
	delete(a.instances, instId)

	xml := fmt.Sprintf(`  <instancesSet>
    <item>
      <instanceId>%s</instanceId>
      <currentState>
        <code>48</code>
        <name>terminated</name>
      </currentState>
    </item>
  </instancesSet>`, instId)

	return []byte(xml), nil
}

func (a *EC2Adapter) StopInstances(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	instId := ctx.ParsedParams["InstanceId.1"]
	if inst, ok := a.instances[instId]; ok {
		inst.State = "stopped"
	}
	xml := fmt.Sprintf(`  <instancesSet><item><instanceId>%s</instanceId><currentState><code>80</code><name>stopped</name></currentState></item></instancesSet>`, instId)
	return []byte(xml), nil
}

func (a *EC2Adapter) StartInstances(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	instId := ctx.ParsedParams["InstanceId.1"]
	if inst, ok := a.instances[instId]; ok {
		inst.State = "running"
	}
	xml := fmt.Sprintf(`  <instancesSet><item><instanceId>%s</instanceId><currentState><code>16</code><name>running</name></currentState></item></instancesSet>`, instId)
	return []byte(xml), nil
}

func (a *EC2Adapter) CreateKeyPair(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	keyName := ctx.ParsedParams["KeyName"]
	if keyName == "" {
		return nil, fmt.Errorf("KeyName is required")
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privDER := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}
	pemData := pem.EncodeToMemory(&privBlock)

	kp := &KeyPairMeta{
		KeyName:        keyName,
		KeyFingerprint: "1f:51:ae:28:bf:89:e4:61:60:f3:84:89:1d:93:31:2e",
		KeyMaterial:    string(pemData),
	}
	a.keyPairs[keyName] = kp

	xml := fmt.Sprintf(`  <keyName>%s</keyName>
  <keyFingerprint>%s</keyFingerprint>
  <keyMaterial>%s</keyMaterial>`, kp.KeyName, kp.KeyFingerprint, kp.KeyMaterial)

	return []byte(xml), nil
}

func (a *EC2Adapter) DescribeKeyPairs(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var items string
	for _, kp := range a.keyPairs {
		items += fmt.Sprintf(`    <item>
      <keyName>%s</keyName>
      <keyFingerprint>%s</keyFingerprint>
    </item>
`, kp.KeyName, kp.KeyFingerprint)
	}

	xml := fmt.Sprintf(`  <keySet>
%s  </keySet>`, items)
	return []byte(xml), nil
}

func (a *EC2Adapter) CreateSecurityGroup(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	grpName := ctx.ParsedParams["GroupName"]
	desc := ctx.ParsedParams["GroupDescription"]
	vpcId := ctx.ParsedParams["VpcId"]
	if vpcId == "" {
		vpcId = "vpc-default"
	}

	grpId := "sg-" + uuid.New().String()[:8]
	a.securityGroups[grpId] = &SecurityGroupMeta{
		GroupId:     grpId,
		GroupName:   grpName,
		Description: desc,
		VpcId:       vpcId,
	}

	xml := fmt.Sprintf(`  <return>true</return>
  <groupId>%s</groupId>`, grpId)
	return []byte(xml), nil
}

func (a *EC2Adapter) DescribeSecurityGroups(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var items string
	for _, sg := range a.securityGroups {
		items += fmt.Sprintf(`    <item>
      <ownerId>%s</ownerId>
      <groupId>%s</groupId>
      <groupName>%s</groupName>
      <groupDescription>%s</groupDescription>
      <vpcId>%s</vpcId>
    </item>
`, ctx.AccountID, sg.GroupId, sg.GroupName, sg.Description, sg.VpcId)
	}

	xml := fmt.Sprintf(`  <securityGroupInfo>
%s  </securityGroupInfo>`, items)
	return []byte(xml), nil
}

func (a *EC2Adapter) AuthorizeSecurityGroupIngress(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	xml := `  <return>true</return>`
	return []byte(xml), nil
}

func (a *EC2Adapter) CreateVpc(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cidr := ctx.ParsedParams["CidrBlock"]
	if cidr == "" {
		cidr = "10.0.0.0/16"
	}
	vpcId := "vpc-" + uuid.New().String()[:8]

	v := &VpcMeta{
		VpcId:     vpcId,
		CidrBlock: cidr,
		State:     "available",
	}
	a.vpcs[vpcId] = v

	xml := fmt.Sprintf(`  <vpc>
    <vpcId>%s</vpcId>
    <state>available</state>
    <cidrBlock>%s</cidrBlock>
    <isDefault>false</isDefault>
  </vpc>`, vpcId, cidr)

	return []byte(xml), nil
}

func (a *EC2Adapter) DescribeVpcs(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var items string
	for _, v := range a.vpcs {
		items += fmt.Sprintf(`    <item>
      <vpcId>%s</vpcId>
      <state>%s</state>
      <cidrBlock>%s</cidrBlock>
      <isDefault>false</isDefault>
    </item>
`, v.VpcId, v.State, v.CidrBlock)
	}

	xml := fmt.Sprintf(`  <vpcSet>
%s  </vpcSet>`, items)
	return []byte(xml), nil
}

func (a *EC2Adapter) CreateSubnet(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	vpcId := ctx.ParsedParams["VpcId"]
	cidr := ctx.ParsedParams["CidrBlock"]
	subnetId := "subnet-" + uuid.New().String()[:8]

	s := &SubnetMeta{
		SubnetId:  subnetId,
		VpcId:     vpcId,
		CidrBlock: cidr,
		State:     "available",
	}
	a.subnets[subnetId] = s

	xml := fmt.Sprintf(`  <subnet>
    <subnetId>%s</subnetId>
    <state>available</state>
    <vpcId>%s</vpcId>
    <cidrBlock>%s</cidrBlock>
  </subnet>`, subnetId, vpcId, cidr)

	return []byte(xml), nil
}

func (a *EC2Adapter) DescribeSubnets(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var items string
	for _, s := range a.subnets {
		items += fmt.Sprintf(`    <item>
      <subnetId>%s</subnetId>
      <state>%s</state>
      <vpcId>%s</vpcId>
      <cidrBlock>%s</cidrBlock>
    </item>
`, s.SubnetId, s.State, s.VpcId, s.CidrBlock)
	}

	xml := fmt.Sprintf(`  <subnetSet>
%s  </subnetSet>`, items)
	return []byte(xml), nil
}

func (a *EC2Adapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Instance", ARNFormat: "arn:aws:ec2:{region}:{account}:instance/{instance-id}", Tags: true, Columns: []string{"InstanceId", "InstanceType", "State", "PublicIp", "LaunchTime"}},
		{Name: "SecurityGroup", ARNFormat: "arn:aws:ec2:{region}:{account}:security-group/{group-id}", Tags: true, Columns: []string{"GroupName", "GroupId", "VpcId"}},
		{Name: "Vpc", ARNFormat: "arn:aws:ec2:{region}:{account}:vpc/{vpc-id}", Tags: true, Columns: []string{"VpcId", "CidrBlock", "State"}},
	}
}

func (a *EC2Adapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "ec2:DescribeInstances", ResourceARNs: []string{"*"}},
		{Name: "ec2:RunInstances", ResourceARNs: []string{"arn:aws:ec2:*:*:instance/*"}},
	}
}

func (a *EC2Adapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *EC2Adapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "RunInstances", Source: "ec2.amazonaws.com"},
	}
}

func (a *EC2Adapter) Metrics() []services.MetricDef {
	return []services.MetricDef{
		{Name: "CPUUtilization", Namespace: "AWS/EC2", Unit: "Percent"},
	}
}

func (a *EC2Adapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *EC2Adapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.instances = make(map[string]*InstanceMeta)
	a.keyPairs = make(map[string]*KeyPairMeta)
	a.securityGroups = make(map[string]*SecurityGroupMeta)
	a.vpcs = make(map[string]*VpcMeta)
	a.subnets = make(map[string]*SubnetMeta)
	return nil
}

func (a *EC2Adapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *EC2Adapter) Restore(data map[string]interface{}) error {
	return nil
}
