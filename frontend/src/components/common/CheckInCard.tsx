import {Button,Card,Space,Tag,Tooltip,Typography} from 'antd';
import {CHECKIN_RESULT_LABELS,CHECKIN_SOURCE_LABELS,CHECKIN_STATUS_LABELS} from '../../constants/checkin';
import type {CheckInResult,MoodCheckIn} from '../../types';

const STATUS_COLOR:Record<MoodCheckIn['status'],string>={pending:'orange',improved:'green',still_troubled:'red',revoked:'default'};

interface CheckInCardProps{
  checkIn:MoodCheckIn;
  // 传入 submitting（当前提交中的回访 id）与 onRespond 时，待回访卡片展示本人确认按钮；compact 只展示关联状态。
  submitting?:number|null;
  onRespond?:(id:number,result:CheckInResult)=>void;
  compact?:boolean;
  title?:string;
}

export function CheckInCard({checkIn,submitting,onRespond,compact,title}:CheckInCardProps){
  const triggerDay=checkIn.trigger_date.slice(0,10);
  const header=(
    <div className="mood-card__top">
      <b>次日回访 · {CHECKIN_STATUS_LABELS[checkIn.status]}</b>
      <Tag color={STATUS_COLOR[checkIn.status]}>{CHECKIN_STATUS_LABELS[checkIn.status]}</Tag>
    </div>
  );
  const meta=(
    <div className="checkin-meta muted">
      来自 {triggerDay} 的低落记录（心情 {checkIn.mood_level}/10）· 首次触发：{CHECKIN_SOURCE_LABELS[checkIn.source]}
    </div>
  );
  if(compact){
    return <Tooltip title={`回访日 ${checkIn.check_in_date.slice(0,10)}`}><Tag color={STATUS_COLOR[checkIn.status]}>{title??'当日回访'} · {CHECKIN_STATUS_LABELS[checkIn.status]}</Tag></Tooltip>;
  }
  return (
    <Card size="small" className="mood-card checkin-card">
      {header}
      {meta}
      {checkIn.status==='pending' && onRespond && (
        <Space className="checkin-actions">
          <Typography.Text type="secondary" className="muted">今天感觉怎么样？回访只能由你本人确认：</Typography.Text>
          <Button type="primary" ghost loading={submitting===checkIn.id} onClick={()=>onRespond(checkIn.id,'improved')}>{CHECKIN_RESULT_LABELS.improved}</Button>
          <Button danger loading={submitting===checkIn.id} onClick={()=>onRespond(checkIn.id,'still_troubled')}>{CHECKIN_RESULT_LABELS.still_troubled}</Button>
        </Space>
      )}
      {(checkIn.status==='improved'||checkIn.status==='still_troubled') && (
        <Typography.Paragraph className="muted" style={{marginBottom:0}}>
          你已于 {checkIn.responded_at?.slice(0,10)} 确认「{CHECKIN_RESULT_LABELS[checkIn.status]}」，该结果不再受后续情绪修改影响。
          {checkIn.result_note?` 留言：${checkIn.result_note}`:''}
        </Typography.Paragraph>
      )}
      {checkIn.status==='revoked' && <Typography.Paragraph className="muted" style={{marginBottom:0}}>情绪指数已回升，这条回访已自动撤销。</Typography.Paragraph>}
    </Card>
  );
}
