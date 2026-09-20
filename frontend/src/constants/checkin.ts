import type {CheckInResult,CheckInSource,CheckInStatus} from '../types';

// 低落情绪阈值与后端 internal/constants/checkin.go 保持一致：心情指数不高于 3 触发次日回访。
export const LOW_MOOD_LEVEL=3;

export const CHECKIN_STATUS_VALUES={PENDING:'pending',IMPROVED:'improved',STILL_TROUBLED:'still_troubled',REVOKED:'revoked'} as const;
export const CHECKIN_STATUS_LABELS:Record<CheckInStatus,string>={pending:'待回访',improved:'已好转',still_troubled:'仍困扰',revoked:'已撤销'};
export const CHECKIN_SOURCE_LABELS:Record<CheckInSource,string>={dashboard:'心情花园',moods:'情绪记录'};
export const CHECKIN_RESULT_LABELS:Record<CheckInResult,string>={improved:'我好转了',still_troubled:'仍在困扰我'};
