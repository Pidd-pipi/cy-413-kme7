export type MoodTag='happy'|'anxious'|'tired'|'angry'|'calm'; export type AssessmentCategory='anxiety'|'depression'|'stress'|'sleep';
export interface User {id:number;email:string;nickname:string;avatar:string;birth_date?:string;gender:string;role:string;created_at:string}
export interface Mood {id:number;user_id:number;mood_level:number;mood_tags:string;note:string;record_date:string;created_at:string}
export interface Assessment {id:number;title:string;description:string;category:AssessmentCategory;questions:string;scoring_rule:string}
export interface Journal {id:number;title:string;content:string;mood_level:number;weather:string;is_private:boolean;created_at:string;updated_at:string}
export interface UserAssessment {id:number;assessment_id:number;score:number;result:string;suggestion:string;created_at:string}
export type FollowUpStatus='pending'|'responded'|'revoked';
export type FollowUpResult='better'|'struggling';
export type FollowUpSource='mood'|'mood_list';
export interface FollowUp {id:number;user_id:number;trigger_date:string;scheduled_date:string;source:FollowUpSource;first_mood_id:number;status:FollowUpStatus;result:string;responded_at?:string;created_at:string;updated_at:string}
export interface ApiResponse<T>{code:number;message:string;data:T}
