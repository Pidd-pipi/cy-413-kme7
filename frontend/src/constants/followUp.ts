import type {FollowUpResult, FollowUpSource, FollowUpStatus} from '../types';

export const FollowUpStatusValues={PENDING:'pending',RESPONDED:'responded',REVOKED:'revoked'} as const;
export const FollowUpResultValues={BETTER:'better',STRUGGLING:'struggling'} as const;
export const FollowUpSourceValues={MOOD:'mood',MOOD_LIST:'mood_list'} as const;

export const FOLLOWUP_STATUS_LABELS:Record<FollowUpStatus,string>={pending:'待回访',responded:'已回访',revoked:'已撤销'};
export const FOLLOWUP_RESULT_LABELS:Record<FollowUpResult,string>={better:'已好转',struggling:'仍困扰'};
export const FOLLOWUP_SOURCE_LABELS:Record<FollowUpSource,string>={mood:'心情花园',mood_list:'情绪记录'};

// 心情指数不高于该值时自动生成次日回访
export const LOW_MOOD_THRESHOLD=3;
