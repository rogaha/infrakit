package group

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/infrakit/plugin/group/util"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
	"sync"
)

// newTestInstancePlugin creates a new instance plugin for use in testing and development.
func newTestInstancePlugin(seedInstances ...instance.Spec) *testplugin {
	plugin := testplugin{idPrefix: util.RandomAlphaNumericString(4), instances: map[instance.ID]instance.Spec{}}
	for _, inst := range seedInstances {
		plugin.addInstance(inst)
	}

	return &plugin
}

type testplugin struct {
	lock      sync.Mutex
	idPrefix  string
	nextID    int
	instances map[instance.ID]instance.Spec
}

func (d *testplugin) instancesCopy() map[instance.ID]instance.Spec {
	d.lock.Lock()
	defer d.lock.Unlock()

	instances := map[instance.ID]instance.Spec{}
	for k, v := range d.instances {
		instances[k] = v
	}
	return instances
}

func (d *testplugin) Validate(req json.RawMessage) error {
	return nil
}

func (d *testplugin) addInstance(spec instance.Spec) instance.ID {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.nextID++
	id := instance.ID(fmt.Sprintf("%s-%d", d.idPrefix, d.nextID))
	d.instances[id] = spec
	return id
}

func (d *testplugin) Provision(spec instance.Spec) (*instance.ID, error) {

	id := d.addInstance(spec)
	return &id, nil
}

func (d *testplugin) Destroy(id instance.ID) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	_, exists := d.instances[id]
	if !exists {
		return errors.New("Instance does not exist")
	}

	delete(d.instances, id)
	return nil
}

func (d *testplugin) DescribeInstances(tags map[string]string) ([]instance.Description, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	desc := []instance.Description{}
	for id, inst := range d.instances {
		allMatched := true
		for searchKey, searchValue := range tags {
			tagValue, has := inst.Tags[searchKey]
			if !has || searchValue != tagValue {
				allMatched = false
			}
		}
		if allMatched {
			desc = append(desc, instance.Description{
				ID:        id,
				LogicalID: inst.LogicalID,
				Tags:      inst.Tags,
			})
		}
	}

	return desc, nil
}

const (
	typeMinion = "minion"
	typeLeader = "leader"
)

type testFlavor struct {
}

type schema struct {
	Type   string
	Size   uint
	Shards []instance.LogicalID
	Init   string
	Tags   map[string]string
}

func (t testFlavor) Validate(flavorProperties json.RawMessage) (flavor.AllocationMethod, error) {

	s := schema{}
	err := json.Unmarshal(flavorProperties, &s)
	if err != nil {
		return flavor.AllocationMethod{}, err
	}

	switch s.Type {
	case typeMinion:
		return flavor.AllocationMethod{Size: s.Size}, nil
	case typeLeader:
		return flavor.AllocationMethod{LogicalIDs: s.Shards}, nil
	default:
		return flavor.AllocationMethod{}, errors.New("Unrecognized node type")
	}
}

func (t testFlavor) Prepare(flavorProperties json.RawMessage, spec instance.Spec) (instance.Spec, error) {
	s := schema{}
	err := json.Unmarshal(flavorProperties, &s)
	if err != nil {
		return spec, err
	}

	spec.Init = s.Init
	for k, v := range s.Tags {
		spec.Tags[k] = v
	}
	return spec, nil
}

func (t testFlavor) Healthy(inst instance.Description) (bool, error) {
	return true, nil
}
