import React, { useEffect, useState } from 'react';
import {
  Activity, BookOpen, Check, ChevronRight, Clock3, Flame, Gamepad2,
  LayoutDashboard, Library, Moon, Plus, RefreshCw, Trophy, X
} from 'lucide-react';
import { MediaShelf, RhythmHub, StudyAssessments } from './components/TrackerViews';

const API = '/api';
const emptyDashboard = { radar_scores: { study_score: 0, sport_score: 0, hobby_score: 0, routine_score: 0 }, today_study_minutes: 0, today_sport_done: false, today_sleep_hours: null, active_streaks: [], upcoming_f1: [], recent_media: [] };
const defaultPomodoro = { seconds: 25 * 60, endAt: null, running: false, category: 'SAT' };

function normalizeDashboard(value) {
  const dashboard = value && typeof value === 'object' ? value : {};
  return {
    ...emptyDashboard,
    ...dashboard,
    radar_scores: { ...emptyDashboard.radar_scores, ...(dashboard.radar_scores || {}) },
    active_streaks: Array.isArray(dashboard.active_streaks) ? dashboard.active_streaks : [],
    upcoming_f1: Array.isArray(dashboard.upcoming_f1) ? dashboard.upcoming_f1 : [],
    recent_media: Array.isArray(dashboard.recent_media) ? dashboard.recent_media : []
  };
}

async function apiFetch(path, options = {}) {
  const token = localStorage.getItem('pixellife_token');
  const response = await fetch(`${API}${path}`, { ...options, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers } });
  if (!response.ok) {
    if (response.status === 401) localStorage.removeItem('pixellife_token');
    const error = new Error(await response.text());
    error.status = response.status;
    throw error;
  }
  return response.status === 204 ? null : response.json();
}

function formatTime(seconds) { return `${Math.floor(seconds / 60).toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`; }
function localJson(key, fallback) { try { return JSON.parse(localStorage.getItem(key)) || fallback; } catch { return fallback; } }
function useToast() { const [toast, setToast] = useState(''); useEffect(() => { if (!toast) return undefined; const timer = setTimeout(() => setToast(''), 3000); return () => clearTimeout(timer); }, [toast]); return [toast, setToast]; }
function restorePomodoro() {
  const saved = localJson('pixellife_pomodoro', defaultPomodoro);
  if (saved.running && !saved.endAt) return { ...saved, running: false, endAt: null };
  if (!saved.running || !saved.endAt) return saved;
  const seconds = Math.max(0, Math.ceil((saved.endAt - Date.now()) / 1000));
  return seconds > 0 ? { ...saved, seconds } : { ...saved, seconds: 25 * 60, endAt: null, running: false };
}

export default function App() {
  const [view, setView] = useState(() => localStorage.getItem('pixellife_view') || 'home');
  const [dashboard, setDashboard] = useState(() => normalizeDashboard(localJson('pixellife_dashboard', emptyDashboard)));
  const [pomodoro, setPomodoro] = useState(restorePomodoro);
  const [toast, setToast] = useToast();
  const [backendOnline, setBackendOnline] = useState(false);

  useEffect(() => {
    const authenticate = async () => {
      if (!localStorage.getItem('pixellife_token')) {
        try { const auth = await apiFetch('/auth/telegram', { method: 'POST', body: JSON.stringify({ telegram_id: 8037087938, username: 'pixel_player' }) }); localStorage.setItem('pixellife_token', auth.token); } catch { /* offline mode */ }
      }
      try { await apiFetch('/health'); setBackendOnline(true); setDashboard(normalizeDashboard(await apiFetch('/dashboard'))); } catch (error) {
        if (error.status === 401) {
          try { const auth = await apiFetch('/auth/telegram', { method: 'POST', body: JSON.stringify({ telegram_id: 8037087938, username: 'pixel_player' }) }); localStorage.setItem('pixellife_token', auth.token); setDashboard(normalizeDashboard(await apiFetch('/dashboard'))); setBackendOnline(true); return; } catch { /* report below */ }
        }
        setBackendOnline(false); setToast('Backend не подключён: данные не отправлены');
      }
    };
    authenticate();
  }, []);

  useEffect(() => { localStorage.setItem('pixellife_view', view); }, [view]);
  useEffect(() => { localStorage.setItem('pixellife_dashboard', JSON.stringify(dashboard)); }, [dashboard]);
  useEffect(() => {
    localStorage.setItem('pixellife_pomodoro', JSON.stringify(pomodoro));
    if (!pomodoro.running) return undefined;
    const tick = () => setPomodoro((current) => {
      const seconds = Math.max(0, Math.ceil((current.endAt - Date.now()) / 1000));
      if (seconds === 0) {
        completeStudy(current.category, setDashboard, setToast);
        return { ...current, seconds: 25 * 60, endAt: null, running: false };
      }
      return { ...current, seconds };
    });
    tick();
    const timer = setInterval(tick, 1000);
    return () => clearInterval(timer);
  }, [pomodoro.running, pomodoro.endAt]);
  useEffect(() => { if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => undefined); }, []);

  const refresh = () => apiFetch('/health').then(() => { setBackendOnline(true); return apiFetch('/dashboard'); }).then(normalizeDashboard).then(setDashboard).then(() => setToast('Синхронизировано')).catch(() => { setBackendOnline(false); setToast('Backend недоступен: локальные данные не потерялись'); });
  return <div className="app-shell">
    <header className="topbar"><div className="brand"><div className="brand-mark">PL</div><div><strong>PIXELLIFE</strong><span>личный ритм-трекер</span></div></div><div className="header-actions"><span className={`sync-dot ${backendOnline ? 'online' : 'offline'}`} title={backendOnline ? 'Backend подключён' : 'Backend не подключён'} /><button className="icon-button" onClick={refresh} aria-label="Обновить"><RefreshCw size={17} /></button></div></header>
    <main className="content">{view === 'home' && <Dashboard dashboard={dashboard} pomodoro={pomodoro} setPomodoro={setPomodoro} setView={setView} setToast={setToast} />}{view === 'study' && <StudyAssessments setToast={setToast} />}{view === 'shelf' && <MediaShelf setDashboard={setDashboard} setToast={setToast} />}{view === 'rhythm' && <RhythmHub dashboard={dashboard} setDashboard={setDashboard} setToast={setToast} />}</main>
    <nav className="bottom-nav">{[[LayoutDashboard, 'home', 'Dashboard'], [BookOpen, 'study', 'Study'], [Library, 'shelf', 'Shelf'], [Activity, 'rhythm', 'Rhythm']].map(([Icon, key, label]) => <button key={key} className={view === key ? 'active' : ''} onClick={() => setView(key)}><Icon size={19} /><span>{label}</span></button>)}</nav>{toast && <div className="toast"><Check size={16} />{toast}</div>}
  </div>;
}

function Dashboard({ dashboard, pomodoro, setPomodoro, setView, setToast }) {
  const [showAdd, setShowAdd] = useState(false);
  return <>
    <section className="hero-row"><div><p className="eyebrow">TODAY · PERSONAL DASHBOARD</p><h1>Your day,<br /><em>your rhythm.</em></h1><p className="hero-copy">Study, body, sleep and the things that keep you moving.</p></div><div className="pixel-status"><span>PIXEL LIFE</span><b>TRACKER ONLINE</b><i /></div></section>
    <PixelCompanion dashboard={dashboard} />
    <section className="dashboard-grid"><article className="panel balance-panel"><div className="panel-heading"><div><span className="kicker">ПОСЛЕДНИЕ 7 ДНЕЙ</span><h2>Баланс жизни</h2></div><span className="status-pill">LIVE</span></div><Radar values={dashboard.radar_scores} /></article><article className="panel focus-panel"><div className="panel-heading"><div><span className="kicker">СЕССИЯ, КОТОРАЯ ЗАСЧИТЫВАЕТСЯ</span><h2>Pomodoro</h2></div><Clock3 size={19} /></div><div className="timer">{formatTime(pomodoro.seconds)}</div><div className="timer-meta"><select value={pomodoro.category} onChange={(event) => setPomodoro({ ...pomodoro, category: event.target.value })}><option>SAT</option><option>IELTS</option><option>NIS</option></select><span>{pomodoro.running ? 'идёт, можно закрыть вкладку' : 'завершение = запись'}</span></div><div className="timer-actions"><button className="primary-button" onClick={() => setPomodoro(pomodoro.running ? { ...pomodoro, running: false, endAt: null } : { ...pomodoro, running: true, endAt: Date.now() + pomodoro.seconds * 1000 })}>{pomodoro.running ? 'Пауза' : 'Старт'} <ChevronRight size={15} /></button><button className="ghost-button" onClick={() => setPomodoro({ ...defaultPomodoro, category: pomodoro.category })}>Сброс</button></div></article></section>
    <section className="section-heading"><div><span className="kicker">БЫСТРАЯ СВОДКА</span><h2>Сегодня</h2></div><button className="text-button" onClick={() => setShowAdd(true)}>Добавить <Plus size={15} /></button></section><section className="stats-grid"><Stat icon={<BookOpen />} label="Учёба" value={`${dashboard.today_study_minutes} мин`} accent="red" /><Stat icon={<Activity />} label="Спорт" value={dashboard.today_sport_done ? 'Сделано' : 'Не начато'} accent="blue" /><Stat icon={<Moon />} label="Сон" value={dashboard.today_sleep_hours ? `${dashboard.today_sleep_hours} ч` : 'Нет записи'} accent="yellow" /></section>
    <section className="section-heading"><div><span className="kicker">РАЗДЕЛЫ</span><h2>Твои линии</h2></div></section><section className="lines-grid"><LineCard icon={<Flame />} title="Хобби и стрики" meta="гитара, кубик, привычки" value={((dashboard.active_streaks || [])[0]?.current_streak) || 0} suffix="дней" accent="orange" onClick={() => setView('rhythm')} /><LineCard icon={<Library />} title="Полка" meta="книги и сериалы" value={(dashboard.recent_media || []).length} suffix="тайтлов" accent="green" onClick={() => setView('shelf')} /><LineCard icon={<Gamepad2 />} title="F1 calendar" meta="важные даты" value={((dashboard.upcoming_f1 || [])[0]?.name) || 'добавить'} suffix="" accent="violet" onClick={() => setView('rhythm')} /></section>{showAdd && <QuickAdd onClose={() => setShowAdd(false)} onStudy={() => { completeStudy('SAT', () => {}, setToast); setShowAdd(false); }} onSport={() => { markSport(setDashboard, setToast); setShowAdd(false); }} />}
  </>;
}

function PixelCompanion({ dashboard }) {
  const active = Number(dashboard.today_study_minutes > 0) + Number(dashboard.today_sport_done) + Number(dashboard.today_sleep_hours >= 7);
  const state = active >= 3 ? 'energized' : active >= 1 ? 'awake' : 'waiting';
  const message = state === 'energized' ? 'Companion is thriving' : state === 'awake' ? 'Keep the run alive' : 'Give your companion a reason to wake up';
  return <section className={`pixel-companion ${state}`}><div className="companion-sprite" aria-hidden="true"><span /><span /><span /></div><div><span className="kicker">PIXEL COMPANION</span><h2>{message}</h2><p>{active}/3 daily signals active</p></div><div className="companion-meter"><i style={{ width: `${Math.round(active / 3 * 100)}%` }} /></div></section>;
}

function StudyView({ dashboard, setDashboard, setToast }) {
  const [sessions, setSessions] = useState([]); const [scores, setScores] = useState([]); const [form, setForm] = useState({ category: 'SAT', duration_minutes: 25, quality_rating: 4, notes: '' }); const [score, setScore] = useState({ category: 'SAT', score: '' });
  const load = () => Promise.all([apiFetch('/study/sessions?limit=20'), apiFetch(`/study/scores/${score.category}`)]).then(([nextSessions, nextScores]) => { setSessions(nextSessions || []); setScores(nextScores || []); }).catch(() => undefined);
  useEffect(() => { load(); }, [score.category]);
  const saveSession = async (event) => { event.preventDefault(); try { await apiFetch('/study/sessions', { method: 'POST', body: JSON.stringify({ ...form, duration_minutes: Number(form.duration_minutes), quality_rating: Number(form.quality_rating) }) }); setDashboard({ ...dashboard, today_study_minutes: dashboard.today_study_minutes + Number(form.duration_minutes) }); setToast('Учебная сессия сохранена'); load(); } catch { setToast('Не сохранено: backend недоступен'); } };
  const saveScore = async (event) => { event.preventDefault(); try { await apiFetch('/study/scores', { method: 'POST', body: JSON.stringify({ ...score, score: Number(score.score) }) }); setToast('Результат добавлен'); load(); } catch { setToast('Не удалось синхронизировать результат'); } };
  return <SectionPage eyebrow="УЧЁБА" title="Время vs результат" intro="Pomodoro засчитывается только после финала. Добавляй баллы, чтобы видеть связь между усилием и ростом."><div className="feature-grid"><form className="form-card" onSubmit={saveSession}><h3>Записать сессию</h3><label>Категория<select value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}><option>SAT</option><option>IELTS</option><option>NISH</option></select></label><label>Минуты<input type="number" min="1" value={form.duration_minutes} onChange={(e) => setForm({ ...form, duration_minutes: e.target.value })} /></label><label>Качество<select value={form.quality_rating} onChange={(e) => setForm({ ...form, quality_rating: e.target.value })}><option value="1">1 / 5</option><option value="3">3 / 5</option><option value="5">5 / 5</option></select></label><label>Заметка<textarea value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder="Что получилось?" /></label><button className="primary-button">Сохранить сессию <Check size={15} /></button></form><form className="form-card" onSubmit={saveScore}><h3>Добавить результат</h3><label>Категория<select value={score.category} onChange={(e) => setScore({ ...score, category: e.target.value })}><option>SAT</option><option>IELTS</option><option>NISH</option></select></label><label>Баллы<input type="number" step="0.1" value={score.score} onChange={(e) => setScore({ ...score, score: e.target.value })} placeholder="например 6.5" required /></label><button className="primary-button">Добавить балл <Plus size={15} /></button><div className="trend-box"><span>Последние результаты</span>{scores.length ? scores.slice(-4).map((item) => <b key={item.id}>{item.score}</b>) : <small>Пока нет данных</small>}</div></form></div><DataList title="История сессий" items={sessions.map((item) => ({ title: item.category, meta: `${item.duration_minutes} мин · качество ${item.quality_rating || '-'}`, note: item.notes }))} empty="Завершённые сессии появятся здесь." /></SectionPage>;
}

function ShelfView({ setDashboard, setToast }) { const [items, setItems] = useState([]); const [form, setForm] = useState({ type: 'book', title: '', status: 'planned', cover_url: '', current_season: '', current_episode: '', notes: '' }); const load = () => apiFetch('/media').then((nextItems) => setItems(Array.isArray(nextItems) ? nextItems : [])).catch(() => setItems([])); useEffect(() => { load(); }, []); const save = async (event) => { event.preventDefault(); try { const item = await apiFetch('/media', { method: 'POST', body: JSON.stringify({ ...form, current_season: form.current_season ? Number(form.current_season) : null, current_episode: form.current_episode ? Number(form.current_episode) : null }) }); setItems((currentItems) => [item, ...currentItems]); setDashboard((dashboard) => ({ ...dashboard, recent_media: [item, ...(dashboard.recent_media || [])].slice(0, 3) })); setForm({ ...form, title: '', notes: '' }); setToast('Добавлено на полку'); } catch { setToast('Не сохранено: backend недоступен'); } }; return <SectionPage eyebrow="ПОЛКА" title="Медиа без хаоса" intro="Книги и сериалы, статус, обложка, сезон, серия и заметки в одном месте."><div className="feature-grid"><form className="form-card" onSubmit={save}><h3>Новый тайтл</h3><label>Тип<select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}><option value="book">Книга</option><option value="series">Сериал</option></select></label><label>Название<input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required placeholder="Название" /></label><label>Статус<select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}><option value="planned">В планах</option><option value="in_progress">В процессе</option><option value="done">Прочитано / просмотрено</option></select></label>{form.type === 'series' && <div className="inline-fields"><label>Сезон<input type="number" min="1" value={form.current_season} onChange={(e) => setForm({ ...form, current_season: e.target.value })} /></label><label>Серия<input type="number" min="1" value={form.current_episode} onChange={(e) => setForm({ ...form, current_episode: e.target.value })} /></label></div>}<label>Обложка URL<input value={form.cover_url} onChange={(e) => setForm({ ...form, cover_url: e.target.value })} placeholder="https://..." /></label><label>Заметки<textarea value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></label><button className="primary-button">Добавить на полку <Plus size={15} /></button></form><div className="list-card"><div className="list-title"><h3>Моя полка</h3><span>{items.length} тайтлов</span></div>{items.length ? items.map((item) => <MediaRow key={item.id} item={item} />) : <EmptyState icon={<Library />} text="Добавь первую книгу или сериал." />}</div></div></SectionPage>; }

function RhythmView({ dashboard, setDashboard, setToast }) { const [sleep, setSleep] = useState({ sleep_hours: '', bedtime: '', wake_time: '' }); const [sportNote, setSportNote] = useState(''); const [streakName, setStreakName] = useState(''); const [streaks, setStreaks] = useState([]); const [event, setEvent] = useState({ name: '', event_date: '', event_type: 'F1' }); useEffect(() => { apiFetch('/hobby/streaks').then((nextStreaks) => setStreaks(Array.isArray(nextStreaks) ? nextStreaks : [])).catch(() => setStreaks([])); }, []); const saveSleep = async (e) => { e.preventDefault(); try { await apiFetch('/sleep/records', { method: 'POST', body: JSON.stringify({ ...sleep, sleep_hours: Number(sleep.sleep_hours) }) }); setDashboard({ ...dashboard, today_sleep_hours: Number(sleep.sleep_hours) }); setToast('Сон записан'); } catch { setToast('Сон сохранён в форме, backend недоступен'); } }; const saveSport = async (e) => { e.preventDefault(); try { await apiFetch('/sport/sessions', { method: 'POST', body: JSON.stringify({ workout_type: 'Домашняя тренировка', completed: true, notes: sportNote }) }); setDashboard({ ...dashboard, today_sport_done: true }); setToast('Тренировка отмечена'); } catch { setToast('Тренировка отмечена локально'); } }; const addStreak = async (e) => { e.preventDefault(); try { const created = await apiFetch('/hobby/streaks', { method: 'POST', body: JSON.stringify({ name: streakName }) }); setStreaks((currentStreaks) => [...currentStreaks, created]); setStreakName(''); setToast('Стрик создан'); } catch { setToast('Не удалось создать стрик'); } }; const completeStreak = async (id) => { try { await apiFetch(`/hobby/streaks/${id}/complete`, { method: 'POST' }); setStreaks((currentStreaks) => currentStreaks.map((item) => item.id === id ? { ...item, current_streak: item.current_streak + 1 } : item)); setToast('День стрика засчитан'); } catch { setToast('Не удалось обновить стрик'); } }; const addEvent = async (e) => { e.preventDefault(); try { await apiFetch('/f1/events', { method: 'POST', body: JSON.stringify(event) }); setDashboard({ ...dashboard, upcoming_f1: [...(dashboard.upcoming_f1 || []), event] }); setEvent({ name: '', event_date: '', event_type: 'F1' }); setToast('Дата добавлена'); } catch { setToast('Не удалось добавить событие'); } }; return <SectionPage eyebrow="РИТМ" title="Тело, сон, хобби" intro="Минимальные отметки, которые помогают видеть реальную неделю, а не идеальную версию себя."><div className="feature-grid"><form className="form-card" onSubmit={saveSleep}><h3><Moon size={18} /> Сон</h3><label>Часов сна<input type="number" min="0" max="24" step="0.1" value={sleep.sleep_hours} onChange={(e) => setSleep({ ...sleep, sleep_hours: e.target.value })} required placeholder="7.5" /></label><div className="inline-fields"><label>Отбой<input type="time" value={sleep.bedtime} onChange={(e) => setSleep({ ...sleep, bedtime: e.target.value })} /></label><label>Подъём<input type="time" value={sleep.wake_time} onChange={(e) => setSleep({ ...sleep, wake_time: e.target.value })} /></label></div><button className="primary-button">Записать сон <Check size={15} /></button></form><form className="form-card" onSubmit={saveSport}><h3><Activity size={18} /> Спорт</h3><p className="form-hint">Одна честная галочка лучше сложного плана.</p><label>Комментарий<textarea value={sportNote} onChange={(e) => setSportNote(e.target.value)} placeholder="Что сделал?" /></label><button className="primary-button" disabled={dashboard.today_sport_done}>{dashboard.today_sport_done ? 'Сегодня готово' : 'Отметить тренировку'} <Check size={15} /></button></form><form className="form-card" onSubmit={addStreak}><h3><Flame size={18} /> Стрики</h3><label>Новое хобби<input value={streakName} onChange={(e) => setStreakName(e.target.value)} required placeholder="Гитара, кубик..." /></label><button className="primary-button">Создать стрик <Plus size={15} /></button><div className="streak-list">{streaks.map((item) => <button type="button" key={item.id} onClick={() => completeStreak(item.id)}><span>{item.name}</span><b>{item.current_streak} дн.</b></button>)}</div></form><form className="form-card" onSubmit={addEvent}><h3><Gamepad2 size={18} /> F1 weekend</h3><label>Событие<input value={event.name} onChange={(e) => setEvent({ ...event, name: e.target.value })} required placeholder="Гран-при Сингапура" /></label><label>Дата<input type="date" value={event.event_date} onChange={(e) => setEvent({ ...event, event_date: e.target.value })} required /></label><button className="primary-button">Добавить дату <Plus size={15} /></button></form></div></SectionPage>; }

async function completeStudy(category, setDashboard, setToast) { try { await apiFetch('/study/sessions', { method: 'POST', body: JSON.stringify({ source: 'pomodoro', category, duration_minutes: 25, quality_rating: 4, notes: 'Pomodoro завершён' }) }); if (setDashboard) setDashboard((dashboard) => ({ ...dashboard, today_study_minutes: dashboard.today_study_minutes + 25 })); setToast('Pomodoro записан в backend'); } catch { setToast('Pomodoro завершён, но не сохранён: backend недоступен'); } }
async function markSport(setDashboard, setToast) { try { await apiFetch('/sport/sessions', { method: 'POST', body: JSON.stringify({ workout_type: 'Домашняя тренировка', completed: true }) }); setDashboard((dashboard) => ({ ...dashboard, today_sport_done: true })); setToast('Тренировка записана в backend'); } catch { setToast('Тренировка не сохранена: backend недоступен'); } }
function SectionPage({ eyebrow, title, intro, children }) { return <section className="section-page"><span className="kicker">{eyebrow}</span><h1>{title}</h1><p className="section-intro">{intro}</p>{children}</section>; }
function Radar({ values }) { const points = [values.study_score, values.sport_score, values.hobby_score, values.routine_score]; const coords = points.map((value, index) => { const angle = (-90 + index * 90) * Math.PI / 180; const radius = 54 * Math.max(value, 8) / 100; return `${50 + Math.cos(angle) * radius},${50 + Math.sin(angle) * radius}`; }).join(' '); return <div className="radar-wrap"><svg viewBox="0 0 100 100" className="radar"><polygon points="50,0 100,50 50,100 0,50" className="radar-grid" /><polygon points="50,18 82,50 50,82 18,50" className="radar-grid" /><polygon points={coords} className="radar-fill" /><polyline points={coords} className="radar-line" /></svg><div className="radar-label top">Учёба <b>{Math.round(points[0])}</b></div><div className="radar-label right">Спорт <b>{Math.round(points[1])}</b></div><div className="radar-label bottom">Рутина <b>{Math.round(points[3])}</b></div><div className="radar-label left">Хобби <b>{Math.round(points[2])}</b></div></div>; }
function Stat({ icon, label, value, accent }) { return <article className={`stat-card ${accent}`}><div className="stat-icon">{icon}</div><div><span>{label}</span><strong>{value}</strong></div></article>; }
function LineCard({ icon, title, meta, value, suffix, accent, onClick }) { return <button className={`line-card ${accent}`} onClick={onClick}><div className="line-icon">{icon}</div><div><span>{meta}</span><h3>{title}</h3><strong>{value} <small>{suffix}</small></strong></div><ChevronRight size={17} className="line-arrow" /></button>; }
function DataList({ title, items, empty }) { return <div className="list-card"><div className="list-title"><h3>{title}</h3><span>{items.length}</span></div>{items.length ? items.map((item, index) => <div className="data-row" key={`${item.title}-${index}`}><div><b>{item.title}</b><span>{item.meta}</span>{item.note && <small>{item.note}</small>}</div><Check size={16} /></div>) : <EmptyState icon={<Clock3 />} text={empty} />}</div>; }
function MediaRow({ item }) { return <div className="data-row media-row">{item.cover_url ? <img src={item.cover_url} alt="" /> : <div className="cover-placeholder">{item.type === 'book' ? 'B' : 'S'}</div>}<div><b>{item.title}</b><span>{item.status === 'in_progress' ? 'В процессе' : item.status === 'done' ? 'Завершено' : 'В планах'}</span>{item.current_season && <small>S{item.current_season} · E{item.current_episode || 0}</small>}</div></div>; }
function EmptyState({ icon, text }) { return <div className="empty-state">{icon}<span>{text}</span></div>; }
function QuickAdd({ onClose, onStudy, onSport }) { return <div className="modal-backdrop" onClick={onClose}><div className="quick-modal" onClick={(event) => event.stopPropagation()}><button className="modal-close" onClick={onClose}><X size={17} /></button><span className="kicker">БЫСТРАЯ ОТМЕТКА</span><h2>Что сделал сегодня?</h2><div className="quick-options"><button onClick={onStudy}><BookOpen size={20} /><span>Учился</span><small>+25 минут</small></button><button onClick={onSport}><Activity size={20} /><span>Тренировался</span><small>домашняя тренировка</small></button><button onClick={onClose}><Moon size={20} /><span>Записать сон</span><small>перейти в раздел Ритм</small></button></div></div></div>; }
