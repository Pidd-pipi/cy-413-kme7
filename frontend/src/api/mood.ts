import {request} from '../utils/request';import type {CheckInSource,Mood,MoodTag} from '../types';
export const listMoods=(date?:string)=>request<Mood[]>(`/moods${date?`?date=${date}`:''}`);
export interface MoodPayload{mood_level:number;mood_tags:MoodTag[];note:string;record_date:string;source?:CheckInSource}
export const saveMood=(payload:MoodPayload)=>request<Mood>('/moods',{method:'POST',body:JSON.stringify(payload)});
export const updateMood=(id:number,payload:MoodPayload)=>request<Mood>(`/moods/${id}`,{method:'PUT',body:JSON.stringify(payload)});
export const deleteMood=(id:number)=>request(`/moods/${id}`,{method:'DELETE'});
