package villain

// ThreatLevel est le niveau de menace d'un vilain.
type ThreatLevel string

const (
	ThreatLow      ThreatLevel = "faible"
	ThreatModerate ThreatLevel = "modere"
	ThreatHigh     ThreatLevel = "eleve"
	ThreatCritical ThreatLevel = "critique"
)

func (level ThreatLevel) Valid() bool {
	switch level {
	case ThreatLow, ThreatModerate, ThreatHigh, ThreatCritical:
		return true
	default:
		return false
	}
}

// Status décrit la situation actuelle d'un vilain.
type Status string

const (
	StatusAtLarge    Status = "en-liberte"
	StatusImprisoned Status = "emprisonne"
)

func (status Status) Valid() bool {
	return status == StatusAtLarge || status == StatusImprisoned
}

const (
	defaultThreatLevel = ThreatLow
	defaultStatus      = StatusAtLarge
)
