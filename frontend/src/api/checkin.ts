import {request} from '../utils/request';
import type {CheckInResult,MoodCheckIn} from '../types';

// 低落情绪回访只能由系统在情绪写操作后生成/撤销，前端只有查询与本人确认两个动作。
export const listCheckIns=(params:{date?:string;triggerDate?:string;status?:string}={})=>{
  const qs=new URLSearchParams();
  if(params.date)qs.set('date',params.date);
  if(params.triggerDate)qs.set('trigger_date',params.triggerDate);
  if(params.status)qs.set('status',params.status);
  const qsString=qs.toString();
  return request<MoodCheckIn[]>(`/checkins${qsString?`?${qsString}`:''}`);
};
export const respondCheckIn=(id:number,result:CheckInResult,note='')=>request<MoodCheckIn>(`/checkins/${id}/respond`,{method:'PUT',body:JSON.stringify({result,note})});
