package level

// --- JSON

type JsonLevel struct {
	Key                 *string             `json:"key,omitempty"`
	Parent              *string             `json:"parent_key,omitempty"`
	Name                *string             `json:"name,omitempty"`
	Rows                *int32              `json:"rows,omitempty"`
	Columns             *int32              `json:"columns,omitempty"`
	Health              *int32              `json:"health_bar,omitempty"`
	Duration            *int32              `json:"duration,omitempty"`
	ComboTimer          *float32            `json:"combo_timer,omitempty"`
	UnitDelayMultiplier *float32            `json:"unit_delay_multiplier,omitempty"`
	MaxActiveUnits      *int32              `json:"max_active_units,omitempty"`
	SpawnsPerSecond     *float32            `json:"spawns_per_second,omitempty"`
	SpawnFrequency      *map[string]float32 `json:"spawn_frequency,omitempty"`
}

// --- Datastore

type datastoreSpawnFrequency struct {
	UnitType       string
	SpawnFrequency float32
}

type DatastoreLevel struct {
	Key    string
	HasKey bool

	Parent    string
	HasParent bool

	Name    string
	HasName bool

	Rows    int32
	HasRows bool

	Columns    int32
	HasColumns bool

	Health    int32
	HasHealth bool

	Duration    int32
	HasDuration bool

	ComboTimer    float32
	HasComboTimer bool

	UnitDelayMultiplier    float32
	HasUnitDelayMultiplier bool

	MaxActiveUnits    int32
	HasMaxActiveUnits bool

	SpawnsPerSecond    float32
	HasSpawnsPerSecond bool

	SpawnFrequency    []datastoreSpawnFrequency
	HasSpawnFrequency bool
}

func (level *DatastoreLevel) MergeParentProperties(parentLevel *DatastoreLevel) {
	if !level.HasName && parentLevel.HasName {
		level.HasName = true
		level.Name = parentLevel.Name
	}

	if !level.HasRows && parentLevel.HasRows {
		level.HasRows = true
		level.Rows = parentLevel.Rows
	}

	if !level.HasColumns && parentLevel.HasColumns {
		level.HasColumns = true
		level.Columns = parentLevel.Columns
	}

	if !level.HasHealth && parentLevel.HasHealth {
		level.HasHealth = true
		level.Health = parentLevel.Health
	}

	if !level.HasDuration && parentLevel.HasDuration {
		level.HasDuration = true
		level.Duration = parentLevel.Duration
	}

	if !level.HasComboTimer && parentLevel.HasComboTimer {
		level.HasComboTimer = true
		level.ComboTimer = parentLevel.ComboTimer
	}

	if !level.HasUnitDelayMultiplier && parentLevel.HasUnitDelayMultiplier {
		level.HasUnitDelayMultiplier = true
		level.UnitDelayMultiplier = parentLevel.UnitDelayMultiplier
	}

	if !level.HasMaxActiveUnits && parentLevel.HasMaxActiveUnits {
		level.HasMaxActiveUnits = true
		level.MaxActiveUnits = parentLevel.MaxActiveUnits
	}

	if !level.HasSpawnsPerSecond && parentLevel.HasSpawnsPerSecond {
		level.HasSpawnsPerSecond = true
		level.SpawnsPerSecond = parentLevel.SpawnsPerSecond
	}

	if !level.HasSpawnFrequency && parentLevel.HasSpawnFrequency {
		level.HasSpawnFrequency = true
		level.SpawnFrequency = parentLevel.SpawnFrequency
	}
}

// --- Conversion

func (level *JsonLevel) ToDatastoreLevel() *DatastoreLevel {
	result := &DatastoreLevel{}

	if level.Key != nil {
		result.Key = *level.Key
		result.HasKey = true
	}

	if level.Parent != nil {
		result.Parent = *level.Parent
		result.HasParent = true
	}

	if level.Name != nil {
		result.Name = *level.Name
		result.HasName = true
	}

	if level.Rows != nil {
		result.Rows = *level.Rows
		result.HasRows = true
	}

	if level.Columns != nil {
		result.Columns = *level.Columns
		result.HasColumns = true
	}

	if level.Health != nil {
		result.Health = *level.Health
		result.HasHealth = true
	}

	if level.Duration != nil {
		result.Duration = *level.Duration
		result.HasDuration = true
	}

	if level.ComboTimer != nil {
		result.ComboTimer = *level.ComboTimer
		result.HasComboTimer = true
	}

	if level.UnitDelayMultiplier != nil {
		result.UnitDelayMultiplier = *level.UnitDelayMultiplier
		result.HasUnitDelayMultiplier = true
	}

	if level.MaxActiveUnits != nil {
		result.MaxActiveUnits = *level.MaxActiveUnits
		result.HasMaxActiveUnits = true
	}

	if level.SpawnsPerSecond != nil {
		result.SpawnsPerSecond = *level.SpawnsPerSecond
		result.HasSpawnsPerSecond = true
	}

	if level.SpawnFrequency != nil {
		for key, value := range *level.SpawnFrequency {
			element := datastoreSpawnFrequency{
				UnitType:       key,
				SpawnFrequency: value,
			}
			result.SpawnFrequency = append(result.SpawnFrequency, element)
		}
		result.HasSpawnFrequency = true
	}

	return result
}

func (level *DatastoreLevel) ToJsonLevel() *JsonLevel {
	result := &JsonLevel{}

	if level.HasKey == true {
		result.Key = new(string)
		*result.Key = level.Key
	}

	if level.HasParent == true {
		result.Parent = new(string)
		*result.Parent = level.Parent
	}

	if level.HasName == true {
		result.Name = new(string)
		*result.Name = level.Name
	}

	if level.HasRows == true {
		result.Rows = new(int32)
		*result.Rows = level.Rows
	}

	if level.HasColumns == true {
		result.Columns = new(int32)
		*result.Columns = level.Columns
	}

	if level.HasHealth == true {
		result.Health = new(int32)
		*result.Health = level.Health
	}

	if level.HasDuration == true {
		result.Duration = new(int32)
		*result.Duration = level.Duration
	}

	if level.HasComboTimer == true {
		result.ComboTimer = new(float32)
		*result.ComboTimer = level.ComboTimer
	}

	if level.HasUnitDelayMultiplier == true {
		result.UnitDelayMultiplier = new(float32)
		*result.UnitDelayMultiplier = level.UnitDelayMultiplier
	}

	if level.HasMaxActiveUnits == true {
		result.MaxActiveUnits = new(int32)
		*result.MaxActiveUnits = level.MaxActiveUnits
	}

	if level.HasSpawnsPerSecond == true {
		result.SpawnsPerSecond = new(float32)
		*result.SpawnsPerSecond = level.SpawnsPerSecond
	}

	if level.HasSpawnFrequency == true {
		spawnFrequency := make(map[string]float32)
		for _, element := range level.SpawnFrequency {
			spawnFrequency[element.UnitType] = element.SpawnFrequency
		}
		result.SpawnFrequency = &spawnFrequency
	}

	return result
}
