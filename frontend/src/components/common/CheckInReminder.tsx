import {Alert,Space} from 'antd';
import {CheckInCard} from './CheckInCard';
import type {CheckInResult,MoodCheckIn} from '../../types';

interface CheckInReminderProps{
  pending:MoodCheckIn[];
  submitting?:number|null;
  onRespond:(id:number,result:CheckInResult)=>void;
}

// CheckInReminder 心情花园与情绪记录共用：在页面顶部提醒今天到期、等待本人确认的回访。
export function CheckInReminder({pending,submitting,onRespond}:CheckInReminderProps){
  if(!pending.length)return null;
  return (
    <Alert
      className="checkin-reminder"
      type="warning"
      showIcon
      message={`有 ${pending.length} 条低落情绪回访在等你确认`}
      description={
        <Space direction="vertical" size={10} style={{width:'100%'}}>
          {pending.map(c=><CheckInCard key={c.id} checkIn={c} submitting={submitting} onRespond={onRespond}/>)}
        </Space>
      }
    />
  );
}
