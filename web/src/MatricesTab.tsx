import React, { useState, useEffect } from 'react';

interface MatrixItem {
  sku_id: string;
  retail_network: string;
  store_format: string;
}

interface SkuRow {
  id: string;
  name: string;
  brand: string;
  category: string;
}

interface ColumnGridRow {
  network: string;
  format: string;
}

interface ManufacturerOption {
  id: number;
  name: string;
}

export const MatricesTab: React.FC = () => {
  const [manufacturers, setManufacturers] = useState<ManufacturerOption[]>([]);
  const [manufacturerId, setManufacturerId] = useState<string>('');
  
  // Динамические списки, загружаемые из базы данных PostgreSQL
  const [skus, setSkus] = useState<SkuRow[]>([]);
  const [columns, setColumns] = useState<ColumnGridRow[]>([]);
  const [activeItems, setActiveItems] = useState<MatrixItem[]>([]);
  
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(false);

  const getHeaders = () => {
    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    const credentials = btoa(`${phone}:${password}`);
    return {
      'Authorization': `Basic ${credentials}`,
      'Content-Type': 'application/json'
    };
  };

  // Загрузка списка всех Производителей FMCG для селектора верхнего уровня
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
        console.error('Ошибка связи с сервером при чтении производителей');
      }
    };
    loadMfg();
  }, []);

  // Каскадная динамическая сборка сетки матриц при выборе Производителя
  useEffect(() => {
    if (!manufacturerId) {
      setSkus([]);
      setColumns([]);
      setActiveItems([]);
      return;
    }

    const loadMatrixData = async () => {
      setLoading(true);
      setMessage('');
      setError('');
      try {
        // 1. Скачиваем из СУБД реальный SKU-лист выбранного клиента через аналитический эндпоинт
        const skuResponse = await fetch(`/api/v1/owner/manufacturers/skus/import?manufacturer_id=${manufacturerId}`, {
          method: 'GET',
          headers: { 'Authorization': getHeaders()['Authorization'] }
        });
        let fetchedSkus: SkuRow[] = [];
        if (skuResponse.ok) {
          const skuData = await skuResponse.json();
          fetchedSkus = skuData || [];
          setSkus(fetchedSkus);
        }

        // 2. Скачиваем адресную программу производителя для вычисления уникальных колонок (Сеть + Формат)
        const storeResponse = await fetch(`/api/v1/owner/manufacturers/stores/import?manufacturer_id=${manufacturerId}`, {
          method: 'GET',
          headers: { 'Authorization': getHeaders()['Authorization'] }
        });
        if (storeResponse.ok) {
          const storeData = await storeResponse.json();
          // storeData содержит массив точек [{retail_network: "X5", store_format: "Магазин у дома", ...}]
          const uniquePairsMap = new Map<string, ColumnGridRow>();
          
          if (Array.isArray(storeData)) {
            storeData.forEach((st: any) => {
              if (st.retail_network && st.store_format) {
                const key = `${st.retail_network}||${st.store_format}`;
                uniquePairsMap.set(key, { network: st.retail_network, format: st.store_format });
              }
            });
          }
          setColumns(Array.from(uniquePairsMap.values()));
        }

        // 3. Скачиваем плоский массив уже сохраненных ранее галочек матрицы дистрибьюции
        const matrixResponse = await fetch(`/api/v1/owner/matrices?manufacturer_id=${manufacturerId}`, {
          method: 'GET',
          headers: getHeaders()
        });
        if (matrixResponse.ok) {
          const matrixData = await matrixResponse.json();
          setActiveItems(matrixData || []);
        }
      } catch (err) {
        setError('Произошла ошибка при сборке динамической конфигурации матрицы с сервера Go');
      } finally {
        setLoading(false);
      }
    };

    loadMatrixData();
  }, [manufacturerId]);
  // Переключение состояния конкретного чекбокса в памяти React
  const handleCheckboxChange = (skuId: string, network: string, format: string) => {
    setActiveItems(prev => {
      const exists = prev.some(item => item.sku_id === skuId && item.retail_network === network && item.store_format === format);
      if (exists) {
        // Если галочка стояла — удаляем строку из массива
        return prev.filter(item => !(item.sku_id === skuId && item.retail_network === network && item.store_format === format));
      } else {
        // Если галочки не было — добавляем новую связь
        return [...prev, { sku_id: skuId, retail_network: network, store_format: format }];
      }
    });
  };

  const isChecked = (skuId: string, network: string, format: string) => {
    return activeItems.some(item => item.sku_id === skuId && item.retail_network === network && item.store_format === format);
  };

  // Отправка матрицы на бэкенд методом полной перезаписи (Full Overwrite на уровне одного Производителя)
  const handleSave = async () => {
    if (!manufacturerId) return;
    setSaving(true);
    setMessage('');
    setError('');

    try {
      const response = await fetch('/api/v1/owner/matrices/save', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          manufacturer_id: Number(manufacturerId),
          items: activeItems
        })
      });

      const data = await response.json();
      if (!response.ok) {
        setError(data.error || 'Ошибка при сохранении матрицы дистрибьюции');
        return;
      }

      setMessage(`Матрица успешно сохранена в PostgreSQL! Записано чекбоксов: ${data.saved_checkboxes_count}`);
    } catch {
      setError('Не удалось отправить пакет данных матрицы на бэкенд');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Управление товарными матрицами дистрибьюции (Сеть + Формат)</h3>
      
      {/* Селектор выбора производителя для каскадного построения сетки по ТЗ */}
      <div style={{ marginBottom: '20px', backgroundColor: '#f8f9fa', padding: '15px', borderRadius: '6px', border: '1px solid #eee' }}>
        <label style={{ marginRight: '10px', fontSize: '14px', fontWeight: 'bold' }}>Выберите производителя:</label>
        <select value={manufacturerId} onChange={e => setManufacturerId(e.target.value)} style={{ padding: '8px 12px', borderRadius: '4px', border: '1px solid #ccc', backgroundColor: '#fff', cursor: 'pointer', fontWeight: 'bold' }}>
          <option value="">-- Выберите из базы данных --</option>
          {manufacturers.map(m => (
            <option key={m.id} value={m.id}>{m.name} (ID: {m.id})</option>
          ))}
        </select>
      </div>

      {message && <div style={{ color: '#137333', backgroundColor: '#e6f4ea', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{message}</div>}
      {error && <div style={{ color: '#c5221f', backgroundColor: '#fce8e6', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{error}</div>}

      {loading ? (
        <p style={{ color: '#666', fontStyle: 'italic' }}>Динамическая сборка сетки матриц из PostgreSQL...</p>
      ) : manufacturerId ? (
        skus.length === 0 || columns.length === 0 ? (
          <div style={{ color: '#777', padding: '20px', backgroundColor: '#f9f9f9', borderRadius: '4px', border: '1px solid #eee' }}>
            ⚠️ Сетка не может быть построена. Убедитесь, что для данного производителя на предыдущем шаге успешно загружен SKU-лист (Шаг Б) и привязана адресная программа торговых точек (Шаг В).
          </div>
        ) : (
          <div style={{ overflowX: 'auto', border: '1px solid #ddd', borderRadius: '6px', backgroundColor: '#fff' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '14px' }}>
              <thead>
                <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                  <th style={{ padding: '12px', borderRight: '1px solid #ddd', minWidth: '220px', backgroundColor: '#f1f3f5' }}>Категория / Бренд / SKU</th>
                  {columns.map((col, idx) => (
                    <th key={idx} style={{ padding: '10px', borderRight: '1px solid #ddd', textAlign: 'center', fontSize: '12px', minWidth: '130px' }}>
                      <div style={{ fontWeight: 'bold', color: '#0275d8' }}>{col.network}</div>
                      <div style={{ color: '#666', fontSize: '11px', marginTop: '3px' }}>{col.format}</div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {skus.map(sku => (
                  <tr key={sku.id} style={{ borderBottom: '1px solid #eee' }}>
                    <td style={{ padding: '12px', borderRight: '1px solid #ddd', fontWeight: '500' }}>
                      <span style={{ color: '#777', fontSize: '11px', display: 'block' }}>{sku.brand} | {sku.category}</span>
                      <strong>{sku.name}</strong> <code style={{ fontSize: '11px', color: '#0275d8' }}>[{sku.id}]</code>
                    </td>
                    {columns.map((col, idx) => (
                      <td key={idx} style={{ padding: '10px', borderRight: '1px solid #ddd', textAlign: 'center' }}>
                        <input 
                          type="checkbox" 
                          checked={isChecked(sku.id, col.network, col.format)} 
                          onChange={() => handleCheckboxChange(sku.id, col.network, col.format)} 
                          style={{ width: '18px', height: '18px', cursor: 'pointer' }} 
                        />
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
            
            <div style={{ padding: '15px', backgroundColor: '#f8f9fa', borderTop: '1px solid #ddd', display: 'flex', justifyContent: 'flex-end' }}>
              <button onClick={handleSave} disabled={saving} style={{ padding: '10px 25px', backgroundColor: '#0275d8', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
                {saving ? 'Сохранение...' : 'Сохранить матрицу дистрибьюции'}
              </button>
            </div>
          </div>
        )
      ) : (
        <p style={{ color: '#999', fontStyle: 'italic' }}>Выберите производителя из списка сверху для отображения сетки распределения матриц.</p>
      )}
    </div>
  );
};
