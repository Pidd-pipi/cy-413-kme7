import {Button, Card, Col, DatePicker, Form, Input, Row, Slider, Typography, message} from 'antd';
import dayjs from 'dayjs';
import {useEffect, useState} from 'react';
import {listMoods, saveMood} from '../api/mood';
import {MoodCard} from '../components/common/MoodCard';
import {MoodSelector} from '../components/common/MoodSelector';
import {EmptyState} from '../components/common/EmptyState';
import {FollowUpCard} from '../components/common/FollowUpCard';
import {useFollowUps} from '../hooks/useFollowUps';
import {LOW_MOOD_THRESHOLD} from '../constants/followUp';
import type {Mood, MoodTag} from '../types';

export function Moods(){
  const [moods,setMoods]=useState<Mood[]>([]);
  const [tags,setTags]=useState<MoodTag[]>(['happy']);
  const [date,setDate]=useState<string>();
  const {followUps,load:loadFollowUps,respond}=useFollowUps();

  const load=()=>listMoods(date).then(setMoods).catch(e=>message.error(e.message));
  useEffect(()=>{load()},[date]);
  const refreshAll=()=>{load();loadFollowUps()};

  const followUpForDate=(d:string)=>followUps.find(f=>f.trigger_date.slice(0,10)===d);
  const groups=Array.from(new Set(moods.map(m=>m.record_date.slice(0,10)))).sort((a,b)=>b.localeCompare(a));

  return <>
    <Typography.Title>情绪记录</Typography.Title>
    <Card className="filter-bar">
      <DatePicker placeholder="按日期筛选" onChange={v=>setDate(v?.format('YYYY-MM-DD'))}/>
      <Button onClick={()=>setDate(undefined)}>清除筛选</Button>
    </Card>
    <Row gutter={[20,20]}>
      <Col xs={24} lg={9}>
        <Card title="新增一条记录">
          <Form layout="vertical" initialValues={{level:6,date:dayjs()}} onFinish={async v=>{
            try{
              const recordDate=v.date.format('YYYY-MM-DD');
              await saveMood({mood_level:v.level,mood_tags:tags,note:v.note||'',record_date:recordDate,source:'mood_list'});
              message.success(v.level<=LOW_MOOD_THRESHOLD?'已保存，低落情绪将在次日回访':'已保存');
              setDate(recordDate);
              refreshAll();
            }catch(e){message.error((e as Error).message)}
          }}>
            <Form.Item name="date" label="日期"><DatePicker style={{width:'100%'}}/></Form.Item>
            <Form.Item name="level" label="心情指数（不高于 3 会触发次日回访）"><Slider min={1} max={10}/></Form.Item>
            <Form.Item label="情绪标签"><MoodSelector value={tags} onChange={setTags}/></Form.Item>
            <Form.Item name="note" label="备注"><Input.TextArea rows={3}/></Form.Item>
            <Button type="primary" htmlType="submit">保存记录</Button>
          </Form>
        </Card>
      </Col>
      <Col xs={24} lg={15}>
        <Card title={`记录列表 (${moods.length})`}>
          {moods.length
            ? <div className="card-list">
                {groups.map(d=>{
                  const followUp=followUpForDate(d);
                  return <div key={d} className="mood-day-group">
                    <div className="mood-day-group__date">
                      <Typography.Text strong>{d}</Typography.Text>
                      {followUp && <FollowUpCard followUp={followUp} onRespond={async(id,result)=>{
                        try{await respond(id,result);message.success('已收到你的回应');refreshAll();}
                        catch(e){message.error((e as Error).message)}
                      }}/>}
                    </div>
                    {moods.filter(m=>m.record_date.slice(0,10)===d).map(m=><MoodCard key={m.id} mood={m}/>)}
                  </div>;
                })}
              </div>
            : <EmptyState title="还没有符合条件的情绪记录"/>}
        </Card>
      </Col>
    </Row>
  </>;
}
