package territory

import "appengine/datastore"

// --- Type definition

type Territory struct {
	Id       *string   `json:"id,omitempty"`
	Sequence *int32    `json:"sequence,omitempty"`
	Name     *string   `json:"name,omitempty"`
	Levels   *[]string `json:"levels"`
}

// --- JSON
// Nothing to do.  The struct attributes handle it all.

// --- Datastore
// Implements PropertyLoadSaver.

type dsTerritory struct {
	Id    string
	HasId bool

	Sequence    int32
	HasSequence bool

	Name    string
	HasName bool

	Levels    []string
	HasLevels bool
}

func (t *Territory) Load(c <-chan datastore.Property) error {
	dst := &dsTerritory{}
	err := datastore.LoadStruct(dst, c)
	if err != nil {
		return err
	}

	if dst.HasId {
		t.Id = new(string)
		*t.Id = dst.Id
	}
	if dst.HasSequence {
		t.Sequence = new(int32)
		*t.Sequence = dst.Sequence
	}
	if dst.HasName {
		t.Name = new(string)
		*t.Name = dst.Name
	}
	if dst.HasLevels {
		t.Levels = &dst.Levels
	}

	return nil
}

func (t *Territory) Save(c chan<- datastore.Property) error {
	dst := &dsTerritory{}

	if t.Id != nil {
		dst.HasId = true
		dst.Id = *t.Id
	}
	if t.Sequence != nil {
		dst.HasSequence = true
		dst.Sequence = *t.Sequence
	}
	if t.Name != nil {
		dst.HasName = true
		dst.Name = *t.Name
	}
	if t.Levels != nil {
		dst.HasLevels = true
		dst.Levels = *t.Levels
	}

	return datastore.SaveStruct(dst, c)
}
