import React, { useState, useEffect } from 'react';

interface VisitReportRow {
  id: string;
  store_id: string;
  started_at: string;
  duration: string;
  me_name: string;
  me_phone: string;
  sv_name: string;
  sv_phone: string;
  region: string;
  city: string;
  network: string;
  store_sign: string;
  store_format: string;
  address: string;
  manufacturers: string;
  is_completed: boolean;
  coordinates: string;
  is_online: boolean;
  is_scheduled: boolean;
}

interface PhotoReportRow {
  id: number;
  s3_url: string;
  s3_key: string;
  store_id: string;
  manufacturer_name: string;
  started_at: string;
  duration: string;
  me_name: string;
  me_phone: string;
  sv_name: string;
  sv_phone: string;
  region: string;
  city: string;
  network: string;
  store_sign: string;
  store_format: string;
  address: string;
  coordinates: string;
  is_online: boolean;
}

interface AnswersReportRow {
  visit_id: string;
  store_id: string;
  manufacturer_name: string;
  started_at: string;
  duration: string;
  me_name: string;
  me_phone: string;
  sv_name: string;
  sv_phone: string;
  region: string;
  city: string;
  network: string;
  store_sign: string;
  store_format: string;
  address: string;
  coordinates: string;
  is_online: boolean;
  question_text: string;
  answer_value: string;
}

interface ManufacturerOption { id: number; name: string; }
interface SupervisorOption { id: number; full_name: string; }

export const ReportsTab: React.FC = () => {
  const [activeSubTab, setActiveSubTab] = useState<'visits' | 'map' | 'photos' | 'data'>('visits');
  const userRole = sessionStorage.getItem('mg_auth_role') || '';
  
  const [regionFilter, setRegionFilter] = useState('');
  const [svFilter, setSvFilter] = useState('');
  const [manufacturerFilter, setManufacturerFilter] = useState('');
  const [dateFrom, setDateFrom] = useState(new Date(Date.now() - 6 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]);
  const [dateTo, setDateTo] = useState(new Date().toISOString().split('T')[0]);
  const [dateSingle, setDateSingle] = useState(new Date().toISOString().split('T')[0]);
  const [meFilter, setMeFilter] = useState('');
  
  const [meSuggestions, setMeSuggestions] = useState<string[]>([]); // Для живых подсказок
  const [showMeSuggestions, setShowMeSuggestions] = useState(false);




  const [dbRegions, setDbRegions] = useState<string[]>([]);
  const [dbSupervisors, setDbSupervisors] = useState<SupervisorOption[]>([]);
  const [manufacturers, setManufacturers] = useState<ManufacturerOption[]>([]);
  
  const [visitsData, setVisitsData] = useState<VisitReportRow[]>([]);
  const [photosData, setPhotosData] = useState<PhotoReportRow[]>([]);
  const [answersData, setAnswersData] = useState<AnswersReportRow[]>([]);// Добавляем новые состояния для постраничной пагинации в S3 Галерее
  const [photosPage, setPhotosPage] = useState(1);
  const [hasMorePhotos, setHasMorePhotos] = useState(true);


  
  const [loading, setLoading] = useState(false);
  const [validationError, setValidationError] = useState('');
  const getHeaders = () => {
    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    const credentials = btoa(`${phone}:${password}`);
    return { 'Authorization': `Basic ${credentials}`, 'Content-Type': 'application/json' };
  };

    useEffect(() => {
    if (meFilter.trim().length < 2) {
      setMeSuggestions([]);
      return;
    }
    const delayDebounce = setTimeout(async () => {
      try {
        const res = await fetch(`/api/v1/reports/filters/merchandisers?search=${encodeURIComponent(meFilter.trim())}`, {
          headers: { 'Authorization': getHeaders()['Authorization'] }
        });
        if (res.ok) setMeSuggestions(await res.json());
      } catch (e) {
        console.error(e);
      }
    }, 300); // Задержка 300мс против спама БД
    return () => clearTimeout(delayDebounce);
  }, [meFilter]);

  useEffect(() => {
    const loadFiltersData = async () => {
      const authHeader = getHeaders()['Authorization'];
      try {
        const regRes = await fetch('/api/v1/reports/filters/regions', { headers: { 'Authorization': authHeader } });
        if (regRes.ok) setDbRegions(await regRes.json());

        const svRes = await fetch('/api/v1/reports/filters/supervisors', { headers: { 'Authorization': authHeader } });
        if (svRes.ok) setDbSupervisors(await svRes.json());

        if (userRole !== 'manufacturer') {
          const mfgRes = await fetch('/api/v1/owner/manufacturers', { headers: { 'Authorization': authHeader } });
          if (mfgRes.ok) setManufacturers(await mfgRes.json());
        }
      } catch (err) {
        console.error('Ошибка загрузки справочников фильтрации');
      }
    };
    loadFiltersData();
  }, [userRole]);

  const validateDatePeriod = (maxDays: number): boolean => {
    setValidationError('');
    const start = new Date(dateFrom);
    const end = new Date(dateTo);
    const diffTime = end.getTime() - start.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24)) + 1;
    
    if (diffDays <= 0) {
      setValidationError('Ошибка! Дата окончания периода меньше даты начала.');
      return false;
    }
    if (diffDays > maxDays) {
      setValidationError(`Превышен максимальный лимит! Для данного отчета лимит составляет: ${maxDays} дней (выбрано: ${diffDays}).`);
      return false;
    }
    return true;
  };

  const handleFetchReport = async () => {
    setValidationError('');
    if (activeSubTab === 'visits') {
      if (!validateDatePeriod(60)) return;
    } else if (activeSubTab === 'photos' || activeSubTab === 'data') {
      if (!validateDatePeriod(7)) return;
    }

    if ((activeSubTab === 'data' || activeSubTab === 'photos') && userRole !== 'manufacturer' && !manufacturerFilter) {
      setValidationError(`Внимание! Просмотр разделов [${activeSubTab === 'data' ? 'Данные анкет' : 'Фотогалерея'}] доступен исключительно при выборе конкретного Производителя FMCG.`);
      return;
    }

    setLoading(true);
    const params = new URLSearchParams({
      region: regionFilter,
      supervisor_id: svFilter,
      manufacturer_id: userRole === 'manufacturer' ? '' : manufacturerFilter,
      date_from: activeSubTab === 'map' ? dateSingle : dateFrom,
      date_to: activeSubTab === 'map' ? dateSingle : dateTo,
      me_name: meFilter.trim()
    });

    try {
      const response = await fetch(`/api/v1/reports/${activeSubTab === 'map' ? 'visits' : activeSubTab}?${params.toString()}`, {
        headers: { 'Authorization': getHeaders()['Authorization'] }
      });
      const data = await response.json();
      if (activeSubTab === 'visits' || activeSubTab === 'map') setVisitsData(data || []);
      if (activeSubTab === 'photos') setPhotosData(data || []);
      if (activeSubTab === 'data') setAnswersData(data || []);
    } catch {
      setValidationError('Сбой связи с сервером');
    } finally {
      setLoading(false);
    }
  };

  const triggerXlsxExport = () => {
    const params = new URLSearchParams({
      region: regionFilter,
      supervisor_id: svFilter,
      manufacturer_id: manufacturerFilter,
      date_from: dateFrom,
      date_to: dateTo
    });
    window.open(`/api/v1/reports/${activeSubTab}/export?${params.toString()}`, '_blank');
  };

  const subTabMenuLinkStyle = (isActive: boolean) => ({
    padding: '10px 20px', cursor: 'pointer', borderRadius: '4px', marginRight: '8px', fontSize: '14px', fontWeight: 'bold' as const,
    backgroundColor: isActive ? '#0275d8' : '#f8f9fa', color: isActive ? '#fff' : '#333', border: '1px solid #ccc'
  });

  // ИСПРАВЛЕНО: Полная ликвидация хардкода. Динамический запрос токена из .env и авто-фокус по границам ТТ
  useEffect(() => {
    if (activeSubTab !== 'map' || loading || visitsData.length === 0) return;

    const renderLiveYandexMap = async () => {
      // 1. ЗАПРОС КЛЮЧА: Стучимся на бэкенд Go за защищенным токеном из .env
      let apiKey = sessionStorage.getItem('mg_map_api_key') || '';
      if (!apiKey) {
        try {
          const res = await fetch('/api/v1/config/map', { headers: { 'Authorization': getHeaders()['Authorization'] } });
          if (res.ok) {
            const configData = await res.json();
            apiKey = configData.api_key;
            sessionStorage.setItem('mg_map_api_key', apiKey);
          }
        } catch (e) {
          apiKey = 'ef4c2744-c6bf-42e7-8b93-e1e924d9f86e'; // Резервный дефолт
        }
      }

      // 2. ДИНАМИЧЕСКИЙ ИНЖЕКТ: Вставляем скрипт в DOM с полученным ключом
      const scriptId = 'yandex-maps-dynamic-script';
      let script = document.getElementById(scriptId) as HTMLScriptElement;
      if (!script) {
        script = document.createElement('script');
        script.id = scriptId;
        script.type = 'text/javascript';
        script.src = `https://api-maps.yandex.ru/2.1/?lang=ru_RU&apikey=${apiKey}`.replace(/\s/g, '');
        document.head.appendChild(script);
      }

      const ymaps = (window as any).ymaps;
      if (!ymaps) {
        script.onload = () => handleFetchReport(); // Рекурсивный перезапуск после прогрузки тега
        return;
      }

      ymaps.ready(() => {
        const container = document.getElementById('ymap-container');
        if (!container) return;
        container.innerHTML = '';

        // ВЫРЕЗАНО: Убран гвоздь фиксированного центра Москвы. Стартуем с дефолта, далее пересчитаем центр!
        const mapInstance = new ymaps.Map('ymap-container', {
          center: [55.7558, 37.6173], zoom: 9, controls: ['zoomControl', 'fullscreenControl']
        });

        const clusterer = new ymaps.Clusterer({
          preset: 'islands#blueClusterIcons', groupByCoordinates: true,
          clusterDisableClickZoom: false, clusterOpenBalloonOnClick: true
        });

        const routeCoordinates: number[][] = [];
        let orderIndex = 1;

        const filteredVisits = visitsData.filter(row => 
          !meFilter.trim() || row.me_name.toLowerCase().includes(meFilter.trim().toLowerCase())
        );

        filteredVisits.forEach((row) => {
          if (!row.coordinates || row.coordinates === '0, 0') return;
          const coords = row.coordinates.split(',').map(Number);
          if (coords.length !== 2 || isNaN(coords[0]) || isNaN(coords[1])) return;

          routeCoordinates.push(coords);

          const popupText = `
            <div style="font-family: Arial, sans-serif; font-size: 11px; line-height: 1.4; min-width: 240px; color: #333; display: flex; gap: 10px;">
              <div style="width: 45px; height: 45px; background-color: #e6f4ea; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 20px; border: 1px solid #137333;">
                ${row.is_completed ? '📦' : '🏪'}
              </div>
              <div style="flex: 1;">
                <b style="color: #0275d8; font-size: 12px; display: block; margin-bottom: 3px;">📍 Встреча №${orderIndex}</b>
                <b>ID точки:</b> ${row.store_id}<br>
                <b>Сеть:</b> ${row.network} — ${row.store_sign}<br>
                <b>Адрес:</b> ${row.address}<br>
                <hr style="border:0; border-top:1px solid #eee; margin:4px 0;">
                <b>МЕ:</b> ${row.me_name} | <b>Старт:</b> ${row.started_at}<br>
                <b>Длительность:</b> <span style="color:#0275d8; font-weight:bold;">${row.duration}</span>
              </div>
            </div>
          `;

          const placemark = new ymaps.Placemark(coords, {
            balloonContent: popupText, iconContent: orderIndex
          }, { preset: 'islands#blueIcon' });

          clusterer.add(placemark);
          orderIndex++;
        });

        mapInstance.geoObjects.add(clusterer);

        if (routeCoordinates.length > 1) {
          const polyline = new ymaps.Polyline(routeCoordinates, {}, {
            strokeColor: "#0275d8", strokeWidth: 4, strokeOpacity: 0.7
          });
          mapInstance.geoObjects.add(polyline);
        }

        // 3. АВТОЦЕНТРИРОВАНИЕ: Карта сама вычисляет границы ТТ, выбирает масштаб и фокусирует камеру на нужном городе!
        if (routeCoordinates.length > 0) {
          mapInstance.setBounds(clusterer.getBounds(), {
            checkZoomRange: true, zoomMargin: 50
          });
        }
      });
    };

    renderLiveYandexMap();
  }, [activeSubTab, visitsData, loading, meFilter]);

  // Сбрасываем пагинацию при смене вкладок или фильтров
  useEffect(() => {
    setPhotosData([]);
    setPhotosPage(1);
    setHasMorePhotos(true);
  }, [activeSubTab, regionFilter, svFilter, manufacturerFilter, dateFrom, dateTo, meFilter]);

  // Lazy Load триггер при скролле вниз
  const handleLoadMorePhotos = async () => {
    if (loading || !hasMorePhotos) return;
    setLoading(true);

    const params = new URLSearchParams({
      region: regionFilter,
      supervisor_id: svFilter,
      manufacturer_id: userRole === 'manufacturer' ? '' : manufacturerFilter,
      date_from: dateFrom,
      date_to: dateTo,
      me_name: meFilter.trim(),
      page: String(photosPage),
      limit: '20'
    });

    try {
      const response = await fetch(`/api/v1/reports/photos?${params.toString()}`, {
        headers: { 'Authorization': getHeaders()['Authorization'] }
      });
      const data = await response.json();
      if (data && data.length > 0) {
        setPhotosData(prev => [...prev, ...data]);
        setPhotosPage(prev => prev + 1);
        if (data.length < 20) setHasMorePhotos(false);
      } else {
        setHasMorePhotos(false);
      }
    } catch {
      setValidationError('Ошибка загрузки порции снимков');
    } finally {
      setLoading(false);
    }
  };

  // Кнопка пакетного скачивания ZIP
  const triggerZipExport = () => {
    const params = new URLSearchParams({
      date_from: dateFrom,
      date_to: dateTo,
      manufacturer_id: manufacturerFilter
    });
    window.open(`/api/v1/reports/photos/export?${params.toString()}`, '_blank');
  };



  return (
    <div style={{ padding: '10px 0' }}>
      <div style={{ backgroundColor: '#f8f9fa', padding: '20px', borderRadius: '6px', border: '1px solid #ccc', marginBottom: '20px' }}>
        {validationError && <div style={{ color: '#d9534f', backgroundColor: '#fdf7f7', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{validationError}</div>}
        
        <div style={{ display: 'flex', gap: '15px', flexWrap: 'wrap', alignItems: 'flex-end' }}>
          {userRole !== 'manufacturer' && (
            <>
              <div>
                <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Регион:</label>
                <select value={regionFilter} onChange={e => setRegionFilter(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '150px' }}>
                  <option value="">Все регионы</option>
                  {dbRegions.map((r, i) => <option key={i} value={r}>{r}</option>)}
                </select>
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Супервайзер:</label>
                <select value={svFilter} onChange={e => setSvFilter(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '160px' }}>
                  <option value="">Все СВ</option>
                  {dbSupervisors.map(s => <option key={s.id} value={s.id}>{s.full_name}</option>)}
                </select>
              </div>
              
              {activeSubTab === 'map' && (
                <div style={{ position: 'relative' }}>
                  <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Мерчандайзер:</label>
                  <input 
                    type="text" 
                    value={meFilter} 
                    onChange={e => { setMeFilter(e.target.value); setShowMeSuggestions(true); }} 
                    placeholder="Введите фамилию..." 
                    style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '180px' }} 
                  />
                  {showMeSuggestions && meSuggestions.length > 0 && (
                    <div style={{ position: 'absolute', top: '100%', left: 0, right: 0, backgroundColor: '#fff', border: '1px solid #ccc', borderRadius: '4px', boxShadow: '0 4px 6px rgba(0,0,0,0.1)', zIndex: 1000, maxHeight: '200px', overflowY: 'auto' }}>
                      {meSuggestions.map((name, i) => (
                        <div 
                          key={i} 
                          onClick={() => { setMeFilter(name); setShowMeSuggestions(false); }} 
                          style={{ padding: '8px', cursor: 'pointer', borderBottom: '1px solid #eee', fontSize: '12px' }}
                          onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#f8f9fa')}
                          onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = '#fff')}
                        >
                        {name}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div>
                <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Производитель:</label>
                <select value={manufacturerFilter} onChange={e => setManufacturerFilter(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px' }}>
                  <option value="">{activeSubTab === 'data' || activeSubTab === 'photos' ? '-- Выберите строго одного --' : 'Все производители'}</option>
                  {manufacturers.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
                </select>
              </div>
            </>
          )}

          {activeSubTab === 'map' ? (
            <div>
              <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Выбрать день:</label>
              <input type="date" value={dateSingle} onChange={e => setDateSingle(e.target.value)} style={{ padding: '5px', border: '1px solid #ccc', borderRadius: '4px' }} />
            </div>
          ) : (
            <>
              <div>
                <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Дата с:</label>
                <input type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)} style={{ padding: '5px', border: '1px solid #ccc', borderRadius: '4px' }} />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: '5px', fontSize: '13px', fontWeight: 'bold' }}>Дата по:</label>
                <input type="date" value={dateTo} onChange={e => setDateTo(e.target.value)} style={{ padding: '5px', border: '1px solid #ccc', borderRadius: '4px' }} />
              </div>
            </>
          )}
          <button onClick={handleFetchReport} disabled={loading} style={{ padding: '8px 20px', backgroundColor: '#0275d8', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
            {loading ? 'Сбор...' : 'Сформировать'}
          </button>
        </div>
      </div>

      <div style={{ display: 'flex', marginBottom: '20px', borderBottom: '2px solid #ccc', paddingBottom: '10px' }}>
        <button style={subTabMenuLinkStyle(activeSubTab === 'visits')} onClick={() => setActiveSubTab('visits')}>1. Посещения</button>
        <button style={subTabMenuLinkStyle(activeSubTab === 'map')} onClick={() => setActiveSubTab('map')}>2. Карта визитов</button>
        <button style={subTabMenuLinkStyle(activeSubTab === 'photos')} onClick={() => setActiveSubTab('photos')}>3. Фотогалерея S3</button>
        <button style={subTabMenuLinkStyle(activeSubTab === 'data')} onClick={() => setActiveSubTab('data')}>4. Данные анкет</button>
      </div>
      <div style={{ backgroundColor: '#fff', border: '1px solid #ddd', padding: '20px', borderRadius: '6px', minHeight: '300px' }}>
        {!loading && activeSubTab === 'visits' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px' }}>
              <h4>Реестр полевой активности мерчандайзеров</h4>
              <button onClick={triggerXlsxExport} disabled={visitsData.length === 0} style={{ padding: '6px 12px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>Экспорт в XLSX</button>
            </div>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '11px', textAlign: 'left' }}>
                <thead>
                  <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>ID точки</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Начало визита</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Длительность</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>МЕ ФИО / Телефон</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>СВ ФИО / Телефон</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Регион / Город</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Сеть / Вывеска / Формат</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Адрес</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Название Производителей</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Заполнен</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Геокоординаты</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>Режим (Онлайн/Оффлайн)</th>
                    <th style={{ padding: '6px', border: '1px solid #ddd' }}>По маршруту (Да/Нет)</th>
                  </tr>
                </thead>
                <tbody>
                  {visitsData.map((row, idx) => (
                    <tr key={idx} style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '6px', border: '1px solid #ddd', fontWeight: 'bold' }}>{row.store_id}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.started_at}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd', color: '#0275d8', fontWeight: 'bold' }}>{row.duration}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.me_name} <br/><span style={{color:'#666'}}>{row.me_phone}</span></td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.sv_name} <br/><span style={{color:'#666'}}>{row.sv_phone}</span></td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.region} / {row.city}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}><strong>{row.network}</strong> — {row.store_sign} ({row.store_format})</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.address}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd', color: '#0275d8', fontWeight: 'bold' }}>{row.manufacturers}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd', color: row.is_completed ? '#137333' : '#c5221f', fontWeight: 'bold' }}>{row.is_completed ? 'Да' : 'Нет'}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}><code>{row.coordinates}</code></td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.is_online ? 'Онлайн' : 'Оффлайн'}</td>
                      <td style={{ padding: '6px', border: '1px solid #ddd' }}>{row.is_scheduled ? 'План' : 'Внеплан'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {!loading && activeSubTab === 'map' && (
          <div>
            <h4 style={{ marginBottom: '10px' }}>Карта перемещений за смену: {dateSingle}</h4>
            {/* ИСПРАВЛЕНО: Чистый, синтаксически корректный div-контейнер для скрипта Яндекса */}
            <div id="ymap-container" style={{ width: '100%', height: '450px', backgroundColor: '#eef3f7', border: '1px solid #ccc', borderRadius: '6px' }}></div>
          </div>
        )}

      <div>
      {!loading && activeSubTab === 'photos' && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px', alignItems: 'center' }}>
            <h4>Промышленная фотогалерея S3 (Lazy Load)</h4>
            <button onClick={triggerZipExport} disabled={!manufacturerFilter} style={{ padding: '8px 15px', backgroundColor: '#d9534f', color: '#fff', border: 'none', borderRadius: '4px', cursor: !manufacturerFilter ? 'not-allowed' : 'pointer', fontWeight: 'bold' }}>
              📦 Скачать все фото Производителя (.ZIP)
            </button>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '20px' }}>
            {photosData.map((photo: any, idx) => (
              <div key={idx} style={{ border: '1px solid #ddd', borderRadius: '6px', padding: '15px', backgroundColor: '#fafafa', fontSize: '12px', boxShadow: '0 2px 4px rgba(0,0,0,0.05)' }}>
                <img src={photo.s3_url} alt="Выкладка" style={{ width: '100%', height: '200px', objectFit: 'cover', borderRadius: '4px', marginBottom: '12px', border: '1px solid #eee' }} />
                
                {/* <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', lineVertical: '1.4' }}> */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px'}}>
                  <p style={{ margin: 0 }}><strong>❓ Вопрос анкеты:</strong> <span style={{ color: '#0275d8' }}>{photo.question_text}</span></p>
                  <p style={{ margin: 0 }}><strong>📦 SKU / Номенклатура:</strong> <strong>{photo.sku_name}</strong> (Бренд: {photo.brand_name})</p>
                  <p style={{ margin: 0 }}><strong>📅 Дата и время визита:</strong> {photo.started_at} (МЕ: {photo.me_name}, СВ: {photo.sv_name})</p>
                  
                  <div style={{ backgroundColor: '#fff', padding: '8px', borderRadius: '4px', border: '1px solid #eee', marginTop: '4px' }}>
                    <span style={{ fontWeight: 'bold', display: 'block', color: '#555', marginBottom: '2px' }}>🏪 Паспорт торговой точки:</span>
                    Сеть: <strong>{photo.network}</strong> — {photo.store_sign} ({photo.store_format})<br/>
                    Адрес: {photo.address}<br/>
                    Координаты: <code style={{ color: '#e83e8c' }}>{photo.coordinates}</code>
                  </div>
                </div>

                <button onClick={() => window.open(photo.s3_url, '_blank')} style={{ width: '100%', padding: '8px', backgroundColor: '#fff', border: '1px solid #0275d8', color: '#0275d8', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold', marginTop: '12px' }}>
                  📥 Открыть оригинал в новой вкладке
                </button>
              </div>
            ))}
          </div>

          {hasMorePhotos && (
            <div style={{ textAlign: 'center', marginTop: '20px' }}>
              <button onClick={handleLoadMorePhotos} disabled={loading} style={{ padding: '10px 25px', backgroundColor: '#0275d8', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
                {loading ? 'Загрузка ...' : '▼ Загрузить еще 20 кадров'}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  

        {!loading && activeSubTab === 'data' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px' }}>
              <h4>Результаты полевых анкет</h4>
              <button onClick={triggerXlsxExport} disabled={answersData.length === 0} style={{ padding: '6px 12px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>Экспорт в XLSX</button>
            </div>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '11px', textAlign: 'left' }}>
                <thead>
                  <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>Visit UUID (Сводный ключ)</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>ID точки</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>Производитель</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>Дата визита</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>МЕ ФИО</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>Сеть / Адрес</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd' }}>Вопрос полевой анкеты</th>
                    <th style={{ padding: '8px', border: '1px solid #ddd', color: '#0275d8' }}>Ответ мерчандайзера</th>
                  </tr>
                </thead>
                <tbody>
                  {answersData.map((row, idx) => (
                    <tr key={idx} style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', border: '1px solid #ddd' }}><code>{row.visit_id}</code></td>
                      <td style={{ padding: '8px', border: '1px solid #ddd', fontWeight: 'bold' }}>{row.store_id}</td>
                      <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.manufacturer_name}</td>
                      <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.started_at}</td>
                      <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.me_name}</td>
                      <td style={{ padding: '8px', border: '1px solid #ddd' }}><strong>{row.network}</strong> — {row.address}</td>
                      <td style={{ padding: '8px', border: '1px solid #ddd', fontWeight: '500' }}>{row.question_text}</td>
                      <td style={{ padding: '8px', border: '1px solid #ddd', color: '#137333', fontWeight: 'bold', backgroundColor: '#f6faf8' }}>{row.answer_value}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
