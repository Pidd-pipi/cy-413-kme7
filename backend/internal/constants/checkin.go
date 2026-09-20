package constants

// LowMoodLevel 是低落情绪阈值：心情指数不高于该值即视为低落，需要生成次日回访。
const LowMoodLevel = 3

const (
	// CheckInStatusPending 待回访：已生成、等待本人在回访日确认。
	CheckInStatusPending = "pending"
	// CheckInStatusImproved 本人确认已好转。
	CheckInStatusImproved = "improved"
	// CheckInStatusStillTroubled 本人确认仍被低落情绪困扰。
	CheckInStatusStillTroubled = "still_troubled"
	// CheckInStatusRevoked 指数回升（当天不再有低落记录）后自动撤销。
	CheckInStatusRevoked = "revoked"
)

// CheckInStatuses 回访结果仅允许本人确认为好转或仍困扰。
var CheckInStatuses = []string{CheckInStatusImproved, CheckInStatusStillTroubled}

const (
	// CheckInSourceMoods 首次触发来源：情绪记录页。
	CheckInSourceMoods = "moods"
	// CheckInSourceDashboard 首次触发来源：心情花园快速记录。
	CheckInSourceDashboard = "dashboard"
)

// CheckInSources 记录首次触发来源时允许的取值；历史数据缺省时按情绪记录页处理。
var CheckInSources = []string{CheckInSourceMoods, CheckInSourceDashboard}
