import {useCallback,useEffect,useMemo,useState} from 'react';
import {listCheckIns,respondCheckIn} from '../api/checkin';
import type {CheckInResult,MoodCheckIn} from '../types';
import dayjs from 'dayjs';

interface UseCheckInsOptions{
  date?:string;
  triggerDate?:string;
  status?:string;
  // auto=false 时不自动请求，由调用方在合适的时机刷新。
  auto?:boolean;
}

// useCheckIns 是心情花园、情绪记录与日记本共享的回访状态来源：
// 所有页面都从同一组接口拉取服务端状态，刷新后页面间状态天然一致。
export function useCheckIns(options:UseCheckInsOptions={}){
  const {date,triggerDate,status,auto=true}=options;
  const [checkIns,setCheckIns]=useState<MoodCheckIn[]>([]);
  const [loading,setLoading]=useState(false);
  const [submitting,setSubmitting]=useState<number|null>(null);
  const load=useCallback(()=>{
    setLoading(true);
    return listCheckIns({date,triggerDate,status})
      .then(setCheckIns)
      .finally(()=>setLoading(false));
  },[date,triggerDate,status]);
  useEffect(()=>{if(auto)load().catch(()=>undefined)},[load,auto]);
  const respond=useCallback(async(id:number,result:CheckInResult,note='')=>{
    setSubmitting(id);
    try{
      const v=await respondCheckIn(id,result,note);
      setCheckIns(prev=>prev.map(item=>item.id===id?v:item));
      return {ok:true as const,data:v};
    }catch(e){
      // 409 表示重复/并发提交与已有结果冲突：刷新为服务端唯一保留的那条结果。
      await load().catch(()=>undefined);
      return {ok:false as const,error:e as Error};
    }finally{
      setSubmitting(null);
    }
  },[load]);
  // 到期及逾期仍待本人确认的回访（回访日不晚于今天）；昨天低落却未确认时，今天仍会被提醒。
  const duePending=useMemo(()=>{
    const today=dayjs().format('YYYY-MM-DD');
    return checkIns.filter(c=>c.status==='pending'&&c.check_in_date.slice(0,10)<=today);
  },[checkIns]);
  return {checkIns,setCheckIns,loading,submitting,load,respond,duePending};
}
