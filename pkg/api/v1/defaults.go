package v1

var _true = t(true)

func SetDefaults_TimerAttrs(obj *TimerAttrs) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_FeatureState(obj *FeatureState) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
}

func SetDefaults_FeatureSpinlocks(obj *FeatureSpinlocks) {
	if obj.Enabled == nil {
		obj.Enabled = _true
	}
	if obj.Enabled == _true && obj.Spinlocks == nil {
		obj.Spinlocks = ui32(4096)
	}
}

func t(v bool) *bool {
	return &v
}

func ui32(v uint32) *uint32 {
	return &v
}
