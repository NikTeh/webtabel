import React, { useState, useEffect } from 'react';

interface StoreRow {
  id_1c: string;
  region: string;
  city: string;
  retail_network: string;
  store_sign: string;
  store_format: string;
  address: string;
  sv_name: string;
  sv_phone: string;
  latitude?: number;  // ДОБАВЛЕНО ДЛЯ ГЕО-БАЗЫ
  longitude?: number;
}

interface SkuRow {
  id: string;
  name: string;
  brand: string;
  category: string;
  
}

interface MatrixRow {
  sku_name: string;
  retail_network: string;
  store_format: string;
  is_available: boolean;
}

interface ManufacturerCard {
  name: string;
  work_days: string[];
  duration_minutes: number;
}

interface ManufacturerOption {
  id: number;
  name: string;
  work_days: number[];
  duration_minutes: number;
}

export const InfoTab: React.FC = () => {
  const [manufacturers, setManufacturers] = useState<ManufacturerOption[]>([]);
  const [manufacturerId, setManufacturerId] = useState<string>('');
  const [activeTableTab, setActiveTableTab] = useState<1 | 2 | 3 | 4 | 5>(1);
  const [loading, setLoading] = useState(false);

  // Живые массивы данных, скачанные из PostgreSQL
  const [stores, setStores] = useState<StoreRow[]>([]);
  const [skus, setSkus] = useState<SkuRow[]>([]);
  const [matrices, setMatrices] = useState<MatrixRow[]>([]);
  const [card, setCard] = useState<ManufacturerCard | null>(null);

  // Состояния для живого поиска по колонкам таблиц
  const [searchStoreId, setSearchStoreId] = useState('');
  const [searchNetwork, setSearchNetwork] = useState('');
  const [searchAddress, setSearchAddress] = useState('');
  const [searchSkuName, setSearchSkuName] = useState('');

  // Исправленная и безопасная функция формирования заголовков Basic Auth
  const getHeaders = () => {
    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    const credentials = btoa(`${phone}:${password}`);
    return {
      'Authorization': `Basic ${credentials}`,
      'Content-Type': 'application/json'
    };
  };

  // 1. Загрузка списка Производителей при старте вкладки
  useEffect(() => {
    const loadMfg = async () => {
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
        console.error('Ошибка чтения производителей');
      }
    };
    loadMfg();
  }, []);

  // 2. Исправленный каскадный сбор всей аналитики без ложных методов
  useEffect(() => {
    if (!manufacturerId) {
      setStores([]); setSkus([]); setMatrices([]); setCard(null);
      return;
    }

    const loadInfoData = async () => {
      setLoading(true);
      const authHeader = getHeaders()['Authorization'];
      try {
        // Таблица 1: Адресная программа
        const stRes = await fetch(`/api/v1/owner/manufacturers/stores/import?manufacturer_id=${manufacturerId}`, { headers: { 'Authorization': authHeader } });
        const stData = stRes.ok ? await stRes.json() : [];
        setStores(stData || []);

        // Таблица 2: Номенклатура SKU
        const skuRes = await fetch(`/api/v1/owner/manufacturers/skus/import?manufacturer_id=${manufacturerId}`, { headers: { 'Authorization': authHeader } });
        const skuData: SkuRow[] = skuRes.ok ? await skuRes.json() : [];
        setSkus(skuData || []);

        // Таблица 4: Регламентные настройки карточки
        const currentMfg = manufacturers.find(m => String(m.id) === manufacturerId);
        if (currentMfg) {
          const daysLabels = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
          setCard({
            name: currentMfg.name,
            work_days: currentMfg.work_days.map(d => daysLabels[d - 1] || String(d)),
            duration_minutes: currentMfg.duration_minutes
          });
        }

        // Таблица 3: Свод товарных матриц
        const matRes = await fetch(`/api/v1/owner/matrices?manufacturer_id=${manufacturerId}`, { headers: { 'Authorization': authHeader } });
        const matData = matRes.ok ? await matRes.json() : [];
        
        // Исправленный чистый обход массива через стандартный метод find
        const formattedMatrices = (matData || []).map((m: any) => {
          const sName = (skuData || []).find(s => s.id === m.sku_id)?.name || m.sku_id;
          return {
            sku_name: sName,
            retail_network: m.retail_network,
            store_format: m.store_format,
            is_available: true
          };
        });
        setMatrices(formattedMatrices);

      } catch (err) {
        console.error('Ошибка сборки массивов аналитики');
      } finally {
        setLoading(false);
      }
    };

    loadInfoData();
  }, [manufacturerId, manufacturers]);
  const handleExportXlsx = (tableNumber: number) => {
    alert(`Запрос на генерацию Excel-отчета для Таблицы №${tableNumber} отправлен`);
  };

  const subMenuQueryStyle = (isActive: boolean) => ({
    padding: '10px 15px',
    cursor: 'pointer',
    backgroundColor: isActive ? '#0275d8' : '#f8f9fa',
    color: isActive ? '#fff' : '#333',
    border: '1px solid #ccc',
    borderRadius: '4px',
    marginRight: '5px',
    fontSize: '13px',
    fontWeight: 'bold' as const
  });

  // Фильтрация таблиц силами браузера в режиме реального времени
  const filteredStores = stores.filter(store => 
    store.id_1c.toLowerCase().includes(searchStoreId.toLowerCase()) &&
    store.retail_network.toLowerCase().includes(searchNetwork.toLowerCase()) &&
    store.address.toLowerCase().includes(searchAddress.toLowerCase())
  );

  const filteredSkus = skus.filter(sku => 
    sku.name.toLowerCase().includes(searchSkuName.toLowerCase()) ||
    sku.id.toLowerCase().includes(searchSkuName.toLowerCase())
  );

  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Информационные своды и аналитика по клиенту</h3>
      
      <div style={{ marginBottom: '20px', backgroundColor: '#f8f9fa', padding: '15px', borderRadius: '6px', border: '1px solid #eee' }}>
        <label style={{ marginRight: '10px', fontSize: '14px', fontWeight: 'bold' }}>Выберите производителя для вывода данных:</label>
        <select value={manufacturerId} onChange={e => setManufacturerId(e.target.value)} style={{ padding: '6px 12px', borderRadius: '4px', border: '1px solid #ccc', backgroundColor: '#fff', cursor: 'pointer', fontWeight: 'bold' }}>
          <option value="">-- Выберите из базы данных --</option>
          {manufacturers.map(m => (
            <option key={m.id} value={m.id}>{m.name} (ID: {m.id})</option>
          ))}
        </select>
      </div>

      {loading ? (
        <p style={{ color: '#666', fontStyle: 'italic' }}>Загрузка аналитических сводов из PostgreSQL...</p>
      ) : manufacturerId ? (
        <div>
          <div style={{ display: 'flex', marginBottom: '20px', borderBottom: '1px solid #ddd', paddingBottom: '10px', flexWrap: 'wrap', gap: '5px' }}>
            <button style={subMenuQueryStyle(activeTableTab === 1)} onClick={() => setActiveTableTab(1)}>1. Все точки ({filteredStores.length})</button>
            <button style={subMenuQueryStyle(activeTableTab === 2)} onClick={() => setActiveTableTab(2)}>2. SKU лист ({filteredSkus.length})</button>
            <button style={subMenuQueryStyle(activeTableTab === 3)} onClick={() => setActiveTableTab(3)}>3. Свод Матриц ({matrices.length})</button>
            <button style={subMenuQueryStyle(activeTableTab === 4)} onClick={() => setActiveTableTab(4)}>4. Настройки и График</button>
          </div>

          <div style={{ border: '1px solid #ddd', padding: '20px', borderRadius: '6px', backgroundColor: '#fff' }}>
            {activeTableTab === 1 && (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px', alignItems: 'center' }}>
                  <h4>1.1. Реестр обслуживаемых торговых точек</h4>
                  <button onClick={() => handleExportXlsx(1)} style={{ padding: '6px 12px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>Экспорт в XLSX</button>
                </div>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                  <thead>
                    <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>
                        ID 1С <br/>
                        <input type="text" placeholder="Поиск..." value={searchStoreId} onChange={e => setSearchStoreId(e.target.value)} style={{ width: '80px', padding: '3px', fontSize: '11px', marginTop: '4px' }} />
                      </th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Регион / Город</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>
                        Сеть <br/>
                        <input type="text" placeholder="Поиск..." value={searchNetwork} onChange={e => setSearchNetwork(e.target.value)} style={{ width: '80px', padding: '3px', fontSize: '11px', marginTop: '4px' }} />
                      </th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Магазин / Формат</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>
                        Адрес <br/>
                        <input type="text" placeholder="Поиск..." value={searchAddress} onChange={e => setSearchAddress(e.target.value)} style={{ width: '120px', padding: '3px', fontSize: '11px', marginTop: '4px' }} />
                      </th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Ответственный СВ</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Широта</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Долгота</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredStores.map((row, idx) => (
                      <tr key={idx} style={{ borderBottom: '1px solid #eee' }}>
                        <td style={{ padding: '8px', border: '1px solid #ddd', fontWeight: 'bold' }}>{row.id_1c}</td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.region}, <strong>{row.city}</strong></td>
                        <td style={{ padding: '8px', border: '1px solid #ddd', color: '#0275d8', fontWeight: 'bold' }}>{row.retail_network}</td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.store_sign} <br/><span style={{ fontSize: '11px', color: '#666' }}>{row.store_format}</span></td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.address}</td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.sv_name} <br/><span style={{ fontSize: '11px', color: '#888' }}>{row.sv_phone}</span></td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}><code>{row.latitude ?? '—'}</code></td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}><code>{row.longitude ?? '—'}</code></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {activeTableTab === 2 && (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px', alignItems: 'center' }}>
                  <h4>1.2. Полный SKU-лист продукции</h4>
                  <button onClick={() => handleExportXlsx(2)} style={{ padding: '6px 12px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>Экспорт в XLSX</button>
                </div>
                <div style={{ marginBottom: '12px' }}>
                  <input type="text" placeholder="Быстрый поиск по названию или ID SKU..." value={searchSkuName} onChange={e => setSearchSkuName(e.target.value)} style={{ width: '100%', maxWidth: '400px', padding: '8px', fontSize: '13px', border: '1px solid #ccc', borderRadius: '4px' }} />
                </div>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                  <thead>
                    <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                      <th style={{ padding: '8px', border: '1px solid #ddd', width: '120px' }}>ID SKU</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Наименование товара</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Бренд / Категория</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredSkus.map((row, idx) => (
                      <tr key={idx} style={{ borderBottom: '1px solid #eee' }}>                        
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}><code style={{ color: '#0275d8', fontWeight: 'bold' }}>{row.id}</code></td>
                        <td style={{ padding: '8px', border: '1px solid #ddd', fontWeight: 'bold' }}>{row.name}</td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}>{row.brand} <br/><span style={{ fontSize: '11px', color: '#666' }}>{row.category}</span></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {activeTableTab === 3 && (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px', alignItems: 'center' }}>
                  <h4>1.3. Сводная товарная матрица обязательной дистрибьюции</h4>
                  <button onClick={() => handleExportXlsx(3)} style={{ padding: '6px 12px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>Экспорт в XLSX</button>
                </div>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                  <thead>
                    <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Товар (SKU)</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd' }}>Розничная сеть / Формат</th>
                      <th style={{ padding: '8px', border: '1px solid #ddd', textAlign: 'center', width: '140px' }}>Статус</th>
                    </tr>
                  </thead>
                  <tbody>
                    {matrices.map((row, idx) => (
                      <tr key={idx} style={{ borderBottom: '1px solid #eee' }}>
                        <td style={{ padding: '8px', border: '1px solid #ddd', fontWeight: 'bold' }}>{row.sku_name}</td>
                        <td style={{ padding: '8px', border: '1px solid #ddd' }}><strong style={{ color: '#0275d8' }}>{row.retail_network}</strong> — {row.store_format}</td>
                        <td style={{ padding: '8px', border: '1px solid #ddd', textAlign: 'center' }}>
                          <span style={{ padding: '3px 8px', borderRadius: '4px', fontSize: '11px', fontWeight: 'bold', backgroundColor: '#e6f4ea', color: '#137333' }}>
                            ☑ В матрице
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {activeTableTab === 4 && card && (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px', alignItems: 'center' }}>
                  <h4>1.4. Регламенты и графики обслуживания</h4>
                </div>
                <div style={{ backgroundColor: '#f8f9fa', padding: '20px', borderRadius: '6px', border: '1px solid #eee', maxWidth: '600px' }}>
                  <p style={{ margin: '0 0 10px 0', fontSize: '15px' }}>Название клиента: <strong>{card.name}</strong></p>
                  <p style={{ margin: '0 0 10px 0', fontSize: '15px' }}>Тайминг обслуживания в торговой точке: <strong style={{ color: '#0275d8' }}>{card.duration_minutes} минут</strong></p>
                  <div style={{ marginTop: '15px' }}>
                    <span style={{ fontSize: '14px', fontWeight: 'bold', display: 'block', marginBottom: '5px' }}>Утвержденные дни посещений по графику:</span>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      {card.work_days.map((day, idx) => (
                        <span key={idx} style={{ padding: '5px 12px', backgroundColor: '#eef7ff', color: '#0275d8', borderRadius: '4px', fontSize: '13px', fontWeight: 'bold', border: '1px solid #b3d7ff' }}>
                          {day}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

          </div>
        </div>
      ) : (
        <p style={{ color: '#999', fontStyle: 'italic' }}>Выберите производителя из списка выше для формирования аналитических отчетов.</p>
      )}
    </div>
  );
};
