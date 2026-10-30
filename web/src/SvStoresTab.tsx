import React, { useState, useEffect } from 'react';

interface ManufacturerSchedule {
  name: string;
  work_days: string;
  duration_minutes: number;
}

interface SvStoreRow {
  id_1c: string;
  region: string;
  city: string;
  retail_network: string;
  store_sign: string;
  store_format: string;
  address: string;
  merchandiser_id: number | null;
  me_full_name: string;
  me_phone: string;
  schedules: ManufacturerSchedule[];
}

interface MerchandiserOption {
  id: number;
  full_name: string;
}

export const SvStoresTab: React.FC = () => {
  const [stores, setStores] = useState<SvStoreRow[]>([]);
  const [merchandisers, setMerchandisers] = useState<MerchandiserOption[]>([]);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const getHeaders = () => {
    const savedPhone = sessionStorage.getItem('mg_auth_phone') || '';
    const savedPassword = sessionStorage.getItem('mg_auth_password') || '';
    return {
      'Authorization': `Basic ${btoa(`${savedPhone}:${savedPassword}`)}`,
      'Content-Type': 'application/json'
    };
  };

  // 1. Загрузка списка торговых точек Супервайзера из PostgreSQL
  const loadStores = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/sv/stores', { 
        method: 'GET', 
        headers: { 'Authorization': getHeaders()['Authorization'] } 
      });
      if (response.ok) {
        const data = await response.json();
        setStores(data || []);
      }
    } catch {
      setError('Ошибка получения адресной программы');
    } finally {
      setLoading(false);
    }
  };

  // 2. Динамическая загрузка только своей команды МЕ для управления вакансиями
  const loadMerchandisers = async () => {
    try {
      const response = await fetch('/api/v1/sv/merchandisers', { 
        method: 'GET', 
        headers: { 'Authorization': getHeaders()['Authorization'] } 
      });
      if (response.ok) {
        const data = await response.json();
        setMerchandisers(data || []);
      }
    } catch {
      console.error('Ошибка загрузки списка мерчандайзеров');
    }
  };

  useEffect(() => {
    loadStores();
    loadMerchandisers();
  }, []);
  // 3. Смена МЕ или перевод точки в статус «Вакансия» силами СВ
  const handleAssignMe = async (storeId1c: string, value: string) => {
    setMessage(''); setError('');
    const parsedId = value === 'vacant' ? null : parseInt(value, 10);

    try {
      const response = await fetch('/api/v1/sv/stores/assign', {
        method: 'POST', 
        headers: getHeaders(),
        body: JSON.stringify({
          store_id_1c: storeId1c,
          merchandiser_id: parsedId
        })
      });
      const data = await response.json();

      if (!response.ok) {
        setError(data.error || 'Ошибка изменения статуса точки');
        return;
      }

      setMessage('Адресная программа успешно обновлена в базе данных!');
      loadStores(); // Мгновенно перерисовываем ФИО и контакты МЕ на экране
    } catch {
      setError('Ошибка при отправке запроса на закрепление сотрудника');
    }
  };

  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Управление адресной программой торговых точек</h3>
      
      {message && <div style={{ color: '#137333', backgroundColor: '#e6f4ea', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{message}</div>}
      {error && <div style={{ color: '#c5221f', backgroundColor: '#fce8e6', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{error}</div>}

      {loading ? (
        <p style={{ color: '#666', fontStyle: 'italic' }}>Загрузка адресной программы из PostgreSQL...</p>
      ) : stores.length === 0 ? (
        <div style={{ color: '#777', padding: '20px', backgroundColor: '#f9f9f9', borderRadius: '4px', border: '1px solid #eee' }}>
          За вашей учетной записью еще не закреплено ни одной торговой точки. Обратитесь к Владельцу для настройки адресной программы.
        </div>
      ) : (
        <div style={{ overflowX: 'auto', border: '1px solid #ddd', borderRadius: '6px', backgroundColor: '#fff' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
            <thead>
              <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                <th style={{ padding: '10px', borderRight: '1px solid #ddd', width: '90px' }}>Код 1С</th>
                <th style={{ padding: '10px', borderRight: '1px solid #ddd', width: '130px' }}>Регион / Город</th>
                <th style={{ padding: '10px', borderRight: '1px solid #ddd', width: '110px' }}>Сеть</th>
                <th style={{ padding: '10px', borderRight: '1px solid #ddd', width: '130px' }}>Магазин / Формат</th>
                <th style={{ padding: '10px', borderRight: '1px solid #ddd' }}>Точный Адрес</th>
                <th style={{ padding: '10px', borderRight: '1px solid #ddd', width: '220px' }}>Графики Производителей</th>
                <th style={{ padding: '10px', textAlign: 'center', width: '200px' }}>Закрепленный МЕ / Статус</th>
              </tr>
            </thead>
            <tbody>
              {stores.map(row => (
                <tr key={row.id_1c} style={{ borderBottom: '1px solid #eee', backgroundColor: row.merchandiser_id === null ? '#fff9f9' : '#fff' }}>
                  <td style={{ padding: '8px', borderRight: '1px solid #ddd' }}><code>{row.id_1c}</code></td>
                  <td style={{ padding: '8px', borderRight: '1px solid #ddd' }}><div style={{ fontSize: '11px', color: '#666' }}>{row.region}</div><strong>{row.city}</strong></td>
                  <td style={{ padding: '8px', borderRight: '1px solid #ddd', fontWeight: 'bold', color: '#0275d8' }}>{row.retail_network}</td>
                  <td style={{ padding: '8px', borderRight: '1px solid #ddd' }}><strong>{row.store_sign}</strong><div style={{ fontSize: '11px', color: '#777', marginTop: '2px' }}>{row.store_format}</div></td>
                  <td style={{ padding: '8px', borderRight: '1px solid #ddd', fontSize: '12px' }}>{row.address}</td>
                  
                  <td style={{ padding: '8px', borderRight: '1px solid #ddd' }}>
                    {row.schedules && row.schedules.length > 0 ? (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                        {row.schedules.map((brand, bIdx) => (
                          <div key={bIdx} style={{ backgroundColor: '#f1f3f5', padding: '4px 6px', borderRadius: '4px', fontSize: '11px', borderLeft: '3px solid #6c757d' }}>
                            <strong>{brand.name}</strong> — {brand.work_days} — <strong>{brand.duration_minutes} мин.</strong>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span style={{ color: '#999', fontSize: '11px', fontStyle: 'italic' }}>Нет привязанных брендов</span>
                    )}
                  </td>

                  <td style={{ padding: '8px', textAlign: 'center' }}>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '5px', alignItems: 'center' }}>
                      <select
                        value={row.merchandiser_id !== null ? row.merchandiser_id : 'vacant'}
                        onChange={(e) => handleAssignMe(row.id_1c, e.target.value)}
                        style={{ padding: '6px', fontSize: '12px', borderRadius: '4px', border: '1px solid #ccc', backgroundColor: row.merchandiser_id === null ? '#fff5f5' : '#fff', color: row.merchandiser_id === null ? '#d9534f' : '#333', cursor: 'pointer', width: '90%' }}
                      >
                        <option value="vacant" style={{ color: '#d9534f', fontWeight: 'bold' }}>ВАКАНСИЯ</option>
                        {merchandisers.map(me => (
                          <option key={me.id} value={me.id}>{me.full_name}</option>
                        ))}
                      </select>
                      {row.merchandiser_id !== null && row.me_phone && (
                        <div style={{ fontSize: '11px', color: '#555', marginTop: '2px' }}>
                          📞 <code>{row.me_phone}</code>
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
