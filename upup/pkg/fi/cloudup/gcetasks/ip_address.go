package gcetasks

import (
	"fmt"

	"github.com/golang/glog"
	"google.golang.org/api/compute/v1"
	"k8s.io/kube-deploy/upup/pkg/fi"
	"k8s.io/kube-deploy/upup/pkg/fi/cloudup/gce"
	"k8s.io/kube-deploy/upup/pkg/fi/cloudup/terraform"
	"k8s.io/kube-deploy/upup/pkg/fi/utils"
)

type IPAddress struct {
	Name    *string
	Address *string

	actual *IPAddress
}

func (d *IPAddress) String() string {
	return utils.JsonString(d)
}

func (d *IPAddress) CompareWithID() *string {
	return d.Name
}

func (e *IPAddress) Find(c *fi.Context) (*IPAddress, error) {
	return e.find(c.Cloud.(*gce.GCECloud))
}

func (e *IPAddress) find(cloud *gce.GCECloud) (*IPAddress, error) {
	r, err := cloud.Compute.Addresses.Get(cloud.Project, cloud.Region, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error listing IPAddresss: %v", err)
	}

	actual := &IPAddress{}
	actual.Address = &r.Address
	actual.Name = &r.Name

	return actual, nil
}

func (e *IPAddress) FindAddress(cloud fi.Cloud) (*string, error) {
	actual, err := e.find(cloud.(*gce.GCECloud))
	if err != nil {
		// TODO: Race here if the address isn't immediately created?
		return nil, fmt.Errorf("error querying for IPAddress: %v", err)
	}
	return actual.Address, nil
}

func (e *IPAddress) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(e, c)
}

func (s *IPAddress) CheckChanges(a, e, changes *IPAddress) error {
	if a != nil {
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
		if changes.Address != nil {
			return fi.CannotChangeField("Address")
		}
	}
	return nil
}

func (_ *IPAddress) RenderGCE(t *gce.GCEAPITarget, a, e, changes *IPAddress) error {
	addr := &compute.Address{
		Name:    *e.Name,
		Address: fi.StringValue(e.Address),
		Region:  t.Cloud.Region,
	}

	if a == nil {
		glog.Infof("GCE creating address: %q", addr.Name)

		_, err := t.Cloud.Compute.Addresses.Insert(t.Cloud.Project, t.Cloud.Region, addr).Do()
		if err != nil {
			return fmt.Errorf("error creating IPAddress: %v", err)
		}
	} else {
		return fmt.Errorf("Cannot apply changes to IPAddress: %v", changes)
	}

	return nil
}

type terraformAddress struct {
	Name *string `json:"name"`
}

func (_ *IPAddress) RenderTerraform(t *terraform.TerraformTarget, a, e, changes *IPAddress) error {
	tf := &terraformAddress{
		Name: e.Name,
	}
	return t.RenderResource("google_compute_address", *e.Name, tf)
}