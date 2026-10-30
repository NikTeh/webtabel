import React, { useState, useEffect } from 'react';

interface ManufacturerOption {
  id: number;
  name: string;
  work_days: number[];
  duration_minutes: number;
}

export const ManufacturersTab: React.FC = () => {
  // Список всех производителей из базы данных для селектора
  const [manufacturers, setManufacturers] = useState<ManufacturerOption[]>([]);
  const [selectedManufacturerId, setSelectedManufacturerId] = useState<string>('');

  // Состояния для формы создания новой карточки
  const [name, setName] = useState('');
  const [duration, setDuration] = useState(0);
  const [selectedDays, setSelectedDays] = useState<number[]>([]);
  
  // Файлы для импорта
  const [skuFile, setSkuFile] = useState<File | null>(null);
  const [storeFile, setStoreFile] = useState<File | null>(null);

  // Статусные сообщения для каждого блока отдельно
  const [cardMsg, setCardMsg] = useState({ text: '', isError: false });
  const [skuMsg, setSkuMsg] = useState({ text: '', isError: false });
  const [storeMsg, setStoreMsg] = useState({ text: '', isError: false });

  const daysOfWeek = [
    { label: 'Пн', value: 1 }, { label: 'Вт', value: 2 }, { label: 'Ср', value: 3 },
    { label: 'Чт', value: 4 }, { label: 'Пт', value: 5 }, { label: 'Сб', value: 6 },
    { label: 'Вс', value: 7 }
  ];

  const getHeaders = () => {
    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    const credentials = btoa(`${phone}:${password}`);
    return {
      'Authorization': `Basic ${credentials}`,
      'Content-Type': 'application/json'
    };
  };

  // Загрузка реального списка клиентов из базы данных PostgreSQL
  const loadManufacturers = async () => {
    try {
      const response = await fetch('/api/v1/owner/manufacturers', {
        method: 'GET',
        headers: { 'Authorization': getHeaders()['Authorization'] }
      });
      if (response.ok) {
        const data = await response.json();
        setManufacturers(data || []);
      }
    } catch {
      console.error('Ошибка подключения к серверу при чтении списка клиентов');
    }
  };

  useEffect(() => {
    loadManufacturers();
  }, []);

  const handleDayToggle = (day: number) => {
    setSelectedDays(prev => 
      prev.includes(day) ? prev.filter(d => d !== day) : [...prev, day]
    );
  };

  // Создание карточки нового клиента
  const handleCreateCard = async (e: React.FormEvent) => {
    e.preventDefault();
    setCardMsg({ text: '', isError: false });

    if (!name.trim() || selectedDays.length === 0 || duration <= 0) {
      setCardMsg({ text: 'Заполните все поля и выберите минимум один день посещения', isError: true });
      return;
    }

    try {
      const response = await fetch('/api/v1/owner/manufacturers', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          name: name.trim(),
          work_days: selectedDays,
          duration_minutes: Number(duration)
        })
      });

      const data = await response.json();
      if (!response.ok) {
        setCardMsg({ text: data.error || 'Ошибка при сохранении карточки Производителя', isError: true });
        return;
      }

      setCardMsg({ text: `Карточка успешно создана! Клиент добавлен в базу.`, isError: false });
      setName('');
      setDuration(0);
      setSelectedDays([]);
      
      // Обновляем список селектора и автоматически выбираем нового клиента
      await loadManufacturers();
      setSelectedManufacturerId(String(data.id));
    } catch {
      setCardMsg({ text: 'Не удалось связаться с сервером бэкенда', isError: true });
    }
  };

  const handleSkuUpload = async () => {
    if (!selectedManufacturerId || !skuFile) return;
    setSkuMsg({ text: '', isError: false }); 
    
    const formData = new FormData();
    formData.append("file", skuFile);

    try {
      // Браузер сам выставит нужный Content-Type и multipart boundary
      const authHeaders = {
        'Authorization': `Basic ${btoa((sessionStorage.getItem('mg_auth_phone') || '') + ':' + (sessionStorage.getItem('mg_auth_password') || ''))}`
      };
      
      const response = await fetch(`/api/v1/owner/manufacturers/skus/import?manufacturer_id=${selectedManufacturerId}`, {
        method: 'POST',
        headers: authHeaders, 
        body: formData       
      });
      
      const data = await response.json();
      if (!response.ok) {
        setSkuMsg({ text: data.error || 'Ошибка импорта SKU', isError: true });
        return;
      }
      setSkuMsg({ text: `Успешно загружено SKU: ${data.inserted}`, isError: false });
      setSkuFile(null);
    } catch {
      setSkuMsg({ text: 'Ошибка отправки файла на сервер', isError: true });
    }
  };


  // Привязка адресной программы (1С кодов точек) к выбранному клиенту
  // const handleStoreUpload = async () => {
  //   if (!selectedManufacturerId || !storeFile) return;
  //   setStoreMsg({ text: '', isError: false });

  //   const formData = new FormData();
  //   formData.append('file', storeFile);

  //   try {
  //     const authHeaders = getHeaders();
  //     delete (authHeaders as any)['Content-Type'];

  //     const response = await fetch(`/api/v1/owner/manufacturers/stores/import?manufacturer_id=${selectedManufacturerId}`, {
  //       method: 'POST',
  //       headers: authHeaders as Record<string, string>, 
  //       body: formData       
  //     });

  //     const data = await response.json();
  //     if (!response.ok) {
  //       setStoreMsg({ text: data.error || 'Ошибка импорта точек', isError: true });
  //       return;
  //     }
  //     setStoreMsg({ text: `Успешно привязано магазинов к данному бренду: ${data.inserted}`, isError: false });
  //     setStoreFile(null);
  //   } catch {
  //     setStoreMsg({ text: 'Ошибка отправки файла на сервер', isError: true });
  //   }
  // };

    const handleStoreUpload = async () => {
    if (!selectedManufacturerId || !storeFile) return;
    setStoreMsg({ text: '', isError: false });

    const formData = new FormData();
    formData.append('file', storeFile);

    try {
      // Браузер сам выставит нужный Content-Type и multipart boundary
      const authHeaders = {
        'Authorization': `Basic ${btoa((sessionStorage.getItem('mg_auth_phone') || '') + ':' + (sessionStorage.getItem('mg_auth_password') || ''))}`
      };
      
      const response = await fetch(`/api/v1/owner/manufacturers/stores/import?manufacturer_id=${selectedManufacturerId}`, {
        method: 'POST',
        headers: authHeaders, 
        body: formData       
      });

      const data = await response.json();
      if (!response.ok) {
        setStoreMsg({ text: data.error || 'Ошибка импорта точек', isError: true });
        return;
      }
      setStoreMsg({ text: `Успешно привязано точек: ${data.inserted}`, isError: false });
      setStoreFile(null);
    } catch {
      setStoreMsg({ text: 'Ошибка отправки файла на сервер', isError: true });
    }
  };


  // Находим параметры текущего выбранного в селекторе производителя
  const currentActiveMfg = manufacturers.find(m => String(m.id) === selectedManufacturerId);

  return (
    <div style={{ padding: '10px 0', display: 'flex', flexDirection: 'column', gap: '30px' }}>
      
      {/* СЕЛЕКТОР ВЫБОРА ПРОИЗВОДИТЕЛЯ ВЕРХНЕГО УРОВНЯ */}
      <div style={{ backgroundColor: '#eef7ff', padding: '15px', borderRadius: '6px', border: '1px solid #b3d7ff' }}>
        <label style={{ marginRight: '15px', fontSize: '14px', fontWeight: 'bold', color: '#004085' }}>
          Выберите Производителя для настройки справочников:
        </label>
        <select 
          value={selectedManufacturerId} 
          onChange={e => {
            setSelectedManufacturerId(e.target.value);
            setSkuMsg({ text: '', isError: false });
            setStoreMsg({ text: '', isError: false });
          }} 
          style={{ padding: '8px 12px', borderRadius: '4px', border: '1px solid #0275d8', fontWeight: 'bold', backgroundColor: '#fff', cursor: 'pointer' }}
        >
          <option value="">-- Выберите из базы данных --</option>
          {manufacturers.map(m => (
            <option key={m.id} value={m.id}>{m.name} (ID: {m.id})</option>
          ))}
        </select>
      </div>

      {/* ШАГ А: СОЗДАНИЕ КАРТОЧКИ */}
      <div>
        <h3>Шаг А: Создание новой карточки Производителя</h3>
        <form onSubmit={handleCreateCard} style={{ backgroundColor: '#f8f9fa', padding: '20px', borderRadius: '6px', border: '1px solid #ccc', maxWidth: '600px' }}>
          {cardMsg.text && (
            <div style={{ color: cardMsg.isError ? '#c5221f' : '#137333', marginBottom: '15px', fontWeight: 'bold' }}>
              {cardMsg.text}
            </div>
          )}
          
                    {/* Исправленная строка разметки: Dumb заменено на стандартный flex-параметр display */}
          <div style={{ display: 'block', marginBottom: '15px' }}>
            {/* <label style={{ display: 'block', marginBottom: '5px', fontSize: '14px', fontWeight: '500' }}>Название клиента (Фабрика FMCG):</label> */}
            <label style={{ display: 'block', marginBottom: '5px', fontSize: '14px', fontWeight: '500' }}>Название клиента (Производитель FMCG):</label>
            <input type="text" value={name} onChange={e => setName(e.target.value)} required placeholder="ООО Марс / Unilever" style={{ width: '100%', padding: '8px', border: '1px solid #ccc', borderRadius: '4px', boxSizing: 'border-box' }} />
          </div>


          <div style={{ marginBottom: '15px' }}>
            <label style={{ display: 'block', marginBottom: '5px', fontSize: '14px', fontWeight: '500' }}>Регламентные дни обслуживания в неделю:</label>
            <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
              {daysOfWeek.map(day => (
                <label key={day.value} style={{ display: 'flex', alignItems: 'center', gap: '3px', cursor: 'pointer', backgroundColor: selectedDays.includes(day.value) ? '#0275d8' : '#eee', color: selectedDays.includes(day.value) ? '#fff' : '#333', padding: '6px 12px', borderRadius: '4px', fontSize: '13px', fontWeight: 'bold' }}>
                  <input type="checkbox" checked={selectedDays.includes(day.value)} onChange={() => handleDayToggle(day.value)} style={{ display: 'none' }} />
                  {day.label}
                </label>
              ))}
            </div>
          </div>

          <div style={{ marginBottom: '20px' }}>
            <label style={{ display: 'block', marginBottom: '5px', fontSize: '14px', fontWeight: '500' }}>Время на визит в торговую точку (минуты):</label>
            <input type="number" value={duration || ''} onChange={e => setDuration(Number(e.target.value))} required placeholder="45" style={{ width: '120px', padding: '8px', border: '1px solid #ccc', borderRadius: '4px' }} />
          </div>

          <button type="submit" style={{ padding: '10px 20px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
            + Создать Производителя
          </button>
        </form>
      </div>

      {/* ШАГИ Б И В: ИМПОРТ ДЛЯ ВЫБРАННОГО КЛИЕНТА */}
      {selectedManufacturerId && currentActiveMfg && (
        <div style={{ display: 'flex', gap: '20px', flexWrap: 'wrap', backgroundColor: '#fafafa', padding: '20px', borderRadius: '8px', border: '1px solid #ddd' }}>
          
          <div style={{ width: '100%', marginBottom: '10px', paddingBottom: '10px', borderBottom: '1px solid #eee' }}>
            <h4 style={{ margin: 0, color: '#333' }}>
              Настройка связанных справочников для: <span style={{ color: '#0275d8', fontSize: '18px' }}>{currentActiveMfg.name}</span>
            </h4>
            <p style={{ margin: '5px 0 0 0', fontSize: '13px', color: '#666' }}>
              Норматив визита: <strong>{currentActiveMfg.duration_minutes} мин.</strong> | Дни недели: <strong>{currentActiveMfg.work_days.map(d => daysOfWeek.find(w => w.value === d)?.label).join(', ')}</strong>
            </p>
          </div>

          {/* Шаг Б: Загрузка SKU */}
          <div style={{ flex: '1', minWidth: '280px', backgroundColor: '#fff', border: '1px dashed #5cb85c', padding: '20px', borderRadius: '6px' }}>
            <h4 style={{ margin: '0 0 10px 0' }}>Шаг Б: Умный импорт SKU листа (Excel)</h4>
            {skuMsg.text && <div style={{ color: skuMsg.isError ? '#c5221f' : '#137333', marginBottom: '12px', fontWeight: 'bold', fontSize: '13px' }}>{skuMsg.text}</div>}
            
            <input 
              type="file" 
              accept=".xlsx" 
              onChange={e => e.target.files && e.target.files[0] && setSkuFile(e.target.files[0])} 
              style={{ display: 'block', marginBottom: '15px', fontSize: '13px' }} 
            />

            <button onClick={handleSkuUpload} disabled={!skuFile} style={{ padding: '8px 15px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: !skuFile ? 'not-allowed' : 'pointer', fontWeight: 'bold' }}>
              Обновить номенклатуру товаров
            </button>
          </div>

          {/* Шаг В: Привязка точек */}
          <div style={{ flex: '1', minWidth: '280px', backgroundColor: '#fff', border: '1px dashed #f0ad4e', padding: '20px', borderRadius: '6px' }}>
            <h4 style={{ margin: '0 0 10px 0' }}>Шаг В: Импорт адресной привязки магазинов</h4>
            {storeMsg.text && <div style={{ color: storeMsg.isError ? '#c5221f' : '#137333', marginBottom: '12px', fontWeight: 'bold', fontSize: '13px' }}>{storeMsg.text}</div>}
            
            <input 
              type="file" 
              accept=".xlsx" 
              onChange={e => e.target.files && e.target.files[0] && setStoreFile(e.target.files[0])} 
              style={{ display: 'block', marginBottom: '15px', fontSize: '13px' }} 
            />

            <button onClick={handleStoreUpload} disabled={!storeFile} style={{ padding: '8px 15px', backgroundColor: '#f0ad4e', color: '#fff', border: 'none', borderRadius: '4px', cursor: !storeFile ? 'not-allowed' : 'pointer', fontWeight: 'bold' }}>
              Привязать магазины из 1С
            </button>
          </div>

        </div>
      )}
    </div>
  );
};
