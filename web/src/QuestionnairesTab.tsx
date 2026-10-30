import React, { useState, useEffect } from 'react';

interface QuestionNode {
  id: string; // Строковый UUID для RAM-сессии React
  db_id?: number; // Системный ID из PostgreSQL (если анкета уже скачана)
  section: 'general' | 'sku';
  sku_id: string | null;
  sku_target_type: 'all' | 'specific' | 'brand' | 'category';
  sku_target_value: string;
  question_text: string;
  response_type: 'text' | 'number' | 'photo' | 'condition' | 'select' | 'date' | 'multiselect';
  available_options: string[]; // Массив распарсенных строк
  options_buffer?: string; // Изолированный текстовый буфер для ввода без багов
  allow_gallery: boolean;
  sort_order: number;
  condition_branch: boolean | null;
  condition_operator: string | null;
  condition_value_text: string[];
  condition_value_numeric: number | null;
  depends_on_question_id: number | string | null; // ИСПРАВЛЕНО: поддержка string/number для предотвращения NaN
  depends_on_operator: string | null;
  depends_on_value: string | null;       // Внешнее проверочное значение
  parent_question_id?: number | null;  
  child_questions: QuestionNode[];  
}

interface ManufacturerOption {
  id: number;
  name: string;
}

interface SkuRow {
  id: string;
  name: string;
  brand: string;
  category: string;
}

export const QuestionnairesTab: React.FC = () => {
  const [manufacturers, setManufacturers] = useState<ManufacturerOption[]>([]);
  const [manufacturerId, setManufacturerId] = useState<string>('');
  const [isAutoCloneEnabled, setIsAutoCloneEnabled] = useState<boolean>(false); // <--- ДОБАВЛЕHО  
  const [activeSubSection, setActiveSubSection] = useState<'general' | 'sku'>('general');
  const [currentVersion, setCurrentVersion] = useState<number>(0);
  
  const [dbSkus, setDbSkus] = useState<SkuRow[]>([]);
  const [dbBrands, setDbBrands] = useState<string[]>([]);
  const [dbCategories, setDbCategories] = useState<string[]>([]);

  const [generalQuestions, setGeneralQuestions] = useState<QuestionNode[]>([]);
  const [skuQuestions, setSkuQuestions] = useState<QuestionNode[]>([]);

  const [skuTargetType, setSkuTargetType] = useState<'all' | 'specific' | 'brand' | 'category'>('all');
  const [skuTargetValue, setSkuTargetValue] = useState<string>('');

  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const getHeaders = () => {
    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    return {
      'Authorization': `Basic ${btoa(`${phone}:${password}`)}`,
      'Content-Type': 'application/json'
    };
  };

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

  useEffect(() => {
    if (!manufacturerId) {
      setDbSkus([]); setDbBrands([]); setDbCategories([]);
      return;
    }
    const loadSkuData = async () => {
      try {
        const response = await fetch(`/api/v1/owner/manufacturers/skus/import?manufacturer_id=${manufacturerId}`, {
          method: 'GET',
          headers: { 'Authorization': getHeaders()['Authorization'] }
        });
        if (response.ok) {
          const data: SkuRow[] = await response.json();
          setDbSkus(data || []);
          const brandsSet = new Set<string>();
          const categoriesSet = new Set<string>();
          (data || []).forEach(item => {
            if (item.brand) brandsSet.add(item.brand);
            if (item.category) categoriesSet.add(item.category);
          });
          setDbBrands(Array.from(brandsSet));
          setDbCategories(Array.from(categoriesSet));
        }
      } catch {
        console.error('Ошибка чтения SKU');
      }
    };
    loadSkuData();
  }, [manufacturerId]);
  // Автоматический сбор текущей активной версии анкеты из PostgreSQL
  useEffect(() => {
    if (!manufacturerId) {
      setGeneralQuestions([]);
      setSkuQuestions([]);
      setCurrentVersion(0);
      setIsAutoCloneEnabled(false); // <--- Сбрасываем при очистке
      return;
    }

    const loadCurrentAnketa = async () => {
      try {
        const response = await fetch(`/api/v1/owner/questionnaires/current?manufacturer_id=${manufacturerId}`, {
          method: 'GET',
          headers: { 'Authorization': getHeaders()['Authorization'] }
        });
        if (response.ok) {
          const data = await response.json();
          setCurrentVersion(data.version || 0);
          setIsAutoCloneEnabled(data.is_auto_clone_enabled || false); // <--- Подтягиваем статус из БД
           if (data.questions && data.questions.length > 0) {
            // 1. Конвертируем плоские строки из БД во временные объекты React RAM
            const flatQuestions: QuestionNode[] = data.questions.map((q: any) => ({
              id: crypto.randomUUID(),
              db_id: q.id,
              section: q.section,
              sku_id: q.sku_id,
              sku_target_type: 'all',
              sku_target_value: '',
              question_text: q.question_text,
              response_type: q.response_type,
              available_options: q.available_options || [],
              options_buffer: q.available_options ? q.available_options.join(' | ') : '',
              allow_gallery: q.allow_gallery,
              sort_order: q.sort_order,
              condition_branch: q.condition_branch,
              condition_operator: q.condition_operator,
              condition_value_text: q.condition_value_text || [],
              condition_value_numeric: q.condition_value_numeric,
              depends_on_question_id: q.depends_on_question_id,
              depends_on_operator: q.depends_on_operator,
              depends_on_value: q.depends_on_value,
              parent_question_id: q.parent_question_id, // Записываем честный ID родителя из PostgreSQL
              child_questions: []
            }));

            // 2. Универсальный рекурсивный сборщик дочерних веток дерева условий
            const buildTree = (parentDbId: number | null, sectionType: 'general' | 'sku'): QuestionNode[] => {
              return flatQuestions
                .filter(q => q.section === sectionType && (parentDbId === null ? !q.parent_question_id : q.parent_question_id === parentDbId))
                .map(q => ({
                  ...q,
                  child_questions: buildTree(q.db_id || null, sectionType)
                }))
                .sort((a, b) => a.sort_order - b.sort_order);
            };

            // Разделяем и рекурсивно собираем дерево по независимым секциям
            setGeneralQuestions(buildTree(null, 'general'));
            setSkuQuestions(buildTree(null, 'sku'));
          }

          else {
            setGeneralQuestions([]);
            setSkuQuestions([]);
          }
        }
      } catch {
        console.error('Ошибка загрузки текущей анкеты');
      }
    };
    loadCurrentAnketa();
  }, [manufacturerId]);

  const getCurrentQuestions = () => {
    return activeSubSection === 'general' ? generalQuestions : skuQuestions;
  };

  const setCurrentQuestions = (updated: QuestionNode[]) => {
    if (activeSubSection === 'general') {
      setGeneralQuestions(updated);
    } else {
      setSkuQuestions(updated);
    }
  };

    // 1. ИСПРАВЛЕНО: Функция рекурсивного обновления полей вопроса в дереве по UUID
  const updateQuestionInTree = (tree: QuestionNode[], id: string, fields: Partial<QuestionNode>): QuestionNode[] => {
    return tree.map(node => {
      if (node.id === id) {
        return { ...node, ...fields };
      }
      if (node.child_questions && node.child_questions.length > 0) {
        return { ...node, child_questions: updateQuestionInTree(node.child_questions, id, fields) };
      }
      return node;
    });
  };

  // 2. ИСПРАВЛЕНО: Функция рекурсивного удаления вопроса из дерева по UUID
  const deleteQuestionFromTree = (tree: QuestionNode[], id: string): QuestionNode[] => {
    return tree
      .filter(node => node.id !== id)
      .map((node, index) => ({
        ...node,
        sort_order: index + 1,
        child_questions: deleteQuestionFromTree(node.child_questions, id)
      }));
  };

  // 3. ИСПРАВЛЕНО: Функция перемещения вопросов стрелочками вверх/вниз по sort_order
  const moveQuestion = (id: string, direction: 'up' | 'down') => {
    const current = getCurrentQuestions();
    const swap = (list: QuestionNode[]): QuestionNode[] => {
      const idx = list.findIndex(q => q.id === id);
      if (idx !== -1) {
        if (direction === 'up' && idx > 0) {
          const updated = [...list];
          [updated[idx - 1], updated[idx]] = [updated[idx], updated[idx - 1]];
          return updated.map((q, i) => ({ ...q, sort_order: i + 1 }));
        }
        if (direction === 'down' && idx < list.length - 1) {
          const updated = [...list];
          [updated[idx], updated[idx + 1]] = [updated[idx + 1], updated[idx]];
          return updated.map((q, i) => ({ ...q, sort_order: i + 1 }));
        }
        return list;
      }
      return list.map(q => ({
        ...q,
        child_questions: swap(q.child_questions)
      }));
    };
    setCurrentQuestions(swap(current));
  };

  //   // МОНОЛИТНЫЙ ВЫЧИСЛИТЕЛЬ: Проверяет характер правок и вызывает нужный эндпоинт Go без дублирования узлов
  // const handleSmartSave = async () => {
  //   if (!manufacturerId) return;
  //   setMessage(''); setError('');

  //   // ИСПРАВЛЕНО ДЛЯ ПРОДАКШН: На сервер должны улетать только корневые вопросы, подвопросы внутри них!
  //   const rootQuestionsPayload = [...generalQuestions, ...skuQuestions];
    
  //   // Собираем плоский массив исключительно для анализа наличия новых вопросов (проверка по db_id)
  //   const allFlatNodesForCheck = tyrannicalFlatten(rootQuestionsPayload);
  //   const hasNewQuestions = allFlatNodesForCheck.some(q => !q.db_id);

  //   if (!hasNewQuestions && currentVersion > 0) {
  //     // Сценарий Б: Косметическое изменение формулировок вопросов
  //     const confirmText = window.confirm(`Вы изменили только текст формулировок. Структура анкеты не нарушена.\nОбновить тексты внутри текущей Версии №${currentVersion} без создания новой?`);
  //     if (!confirmText) return;

  //     setSaving(true);
  //     try {
  //       const payload = allFlatNodesForCheck.map(q => ({
  //         id: q.db_id,
  //         question_text: q.question_text
  //       }));

  //       const response = await fetch('/api/v1/owner/questionnaires/update-text', {
  //         method: 'PUT',
  //         headers: getHeaders(),
  //         body: JSON.stringify(payload)
  //       });
  //       if (response.ok) {
  //         setMessage('Формулировки вопросов успешно обновлены внутри текущей версии!');
  //       } else {
  //         setError('Ошибка обновления текстов на сервере Go');
  //       }
  //     } catch {
  //       setError('Сбой сети при обновлении текстов');
  //     } finally {
  //       setSaving(false);
  //     }
  //   } else {
  //     // Сценарий А: Структурное изменение — Создание новой версии
  //     const nextVer = currentVersion + 1;
  //     const confirmStructure = window.confirm(`Внимание! Обнаружены изменения в структуре анкеты (добавлены новые вопросы или изменены типы).\nБудет создана новая Версия №${nextVer}. Вы уверены?`);
  //     if (!confirmStructure) return;

  //     setSaving(true);
  //     try {
  //       const response = await fetch('/api/v1/owner/questionnaires/save', {
  //         method: 'POST',
  //         headers: getHeaders(),
  //         // ИСПРАВЛЕНО: передаем строго структурированное дерево корневых элементов
  //         body: JSON.stringify({
  //           manufacturer_id: Number(manufacturerId),
  //           is_auto_clone_enabled: isAutoCloneEnabled, // <--- Передаем флаг в JSON пакете            
  //           questions: rootQuestionsPayload
  //         })
  //       });
  //       const data = await response.json();
  //       if (response.ok) {
  //         setMessage(`Новая структура успешно утверждена! Создана Версия №${data.version || nextVer}`);
  //         setCurrentVersion(data.version || nextVer);
  //         setGeneralQuestions([]);
  //         setSkuQuestions([]);
  //       } else {
  //         setError(data.error || 'Ошибка сохранения новой версии');
  //       }
  //     } catch {
  //       setError('Сбой сети при создании новой версии');
  //     } finally {
  //       setSaving(false);
  //     }
  //   }
  // };

    const handleSmartSave = async () => {
    if (!manufacturerId) return;
    setMessage(''); setError('');

    // ИСПРАВЛЕНО ДЛЯ ПРОДАКШН: На сервер должны улетать только корневые вопросы, подвопросы внутри них!
    const rootQuestionsPayload = [...generalQuestions, ...skuQuestions];
    
    // Собираем плоский массив исключительно для анализа наличия новых вопросов (проверка по db_id)
    const allFlatNodesForCheck = tyrannicalFlatten(rootQuestionsPayload);
    const hasNewQuestions = allFlatNodesForCheck.some(q => !q.db_id);

    if (!hasNewQuestions && currentVersion > 0) {
      // Сценарий Б: Косметическое изменение формулировок вопросов без изменения структуры
      const confirmText = window.confirm(`Вы изменили только текст формулировок. Структура анкеты не нарушена.\nОбновить тексты внутри текущей Версии №${currentVersion} без создания новой?`);
      if (!confirmText) return;

      setSaving(true);
      try {
        const payload = allFlatNodesForCheck.map(q => ({
          id: q.db_id,
          question_text: q.question_text
        }));

        const response = await fetch('/api/v1/owner/questionnaires/update-text', {
          method: 'PUT',
          headers: getHeaders(),
          body: JSON.stringify(payload)
        });
        if (response.ok) {
          setMessage('Формулировки вопросов успешно обновлены внутри текущей версии!');
        } else {
          setError('Ошибка обновления текстов на сервере Go');
        }
      } catch {
        setError('Сбой сети при обновлении текстов');
      } finally {
        setSaving(false);
      }
    } else {
      // Сценарий А: Структурное изменение — Создание новой версии анкеты
      const nextVer = currentVersion + 1;
      const confirmStructure = window.confirm(`Внимание! Обнаружены изменения в структуре анкеты (добавлены новые вопросы или изменены типы).\nБудет создана новая Версия №${nextVer}. Вы уверены?`);
      if (!confirmStructure) return;

      // СТРОГИЙ ПРОМЫШЛЕННЫЙ ПЕРЕМАППИНГ: изолируем depends_on_question_id от гонки версий и старых ID СУБД
      const dbIdToUuidMap = new Map<number, string>();
      allFlatNodesForCheck.forEach(q => {
        if (q.db_id) {
          dbIdToUuidMap.set(q.db_id, q.id);
        }
      });

      // Рекурсивная функция очистки и подготовки дерева перед отправкой в Go бэкенд
      const cleanTreeOptions = (nodes: QuestionNode[]): any[] => {
        return nodes.map(n => {
          let finalDependsId = n.depends_on_question_id;
          
          // Если зависимость указана как старое число из СУБД, подменяем на UUID текущей RAM-сессии
          if (typeof finalDependsId === 'number' && dbIdToUuidMap.has(finalDependsId)) {
            finalDependsId = dbIdToUuidMap.get(finalDependsId)!;
          }
          
          return {
            id: n.id,
            section: n.section,
            sku_id: n.sku_id,
            question_text: n.question_text,
            response_type: n.response_type,
            available_options: n.available_options,
            allow_gallery: n.allow_gallery,
            sort_order: n.sort_order,
            condition_branch: n.condition_branch,
            condition_operator: n.condition_operator,
            condition_value_text: n.condition_value_text,
            condition_value_numeric: n.condition_value_numeric,
            depends_on_question_id: finalDependsId, // Улетает чистый, безопасный UUID текущего дерева
            depends_on_operator: n.depends_on_operator,
            depends_on_value: n.depends_on_value,
            child_questions: cleanTreeOptions(n.child_questions)
          };
        });
      };

      const sanitizedGeneral = cleanTreeOptions(generalQuestions);
      const sanitizedSku = cleanTreeOptions(skuQuestions);

      setSaving(true);
      try {
        const response = await fetch('/api/v1/owner/questionnaires/save', {
          method: 'POST',
          headers: getHeaders(),
          body: JSON.stringify({
            manufacturer_id: Number(manufacturerId),
            is_auto_clone_enabled: isAutoCloneEnabled,            
            questions: [...sanitizedGeneral, ...sanitizedSku] // Передаем строго очищенное дерево без конфликтов версий
          })
        });
        
        const data = await response.json();
        if (response.ok) {
          setMessage(`Новая структура успешно утверждена! Создана Версия №${data.version || nextVer}`);
          setCurrentVersion(data.version || nextVer);
          setGeneralQuestions([]);
          setSkuQuestions([]);
        } else {
          setError(data.error || 'Ошибка сохранения новой версии');
        }
      } catch {
        setError('Сбой сети при создании новой версии');
      } finally {
        setSaving(false);
      }
    }
  };



  const tyrannicalFlatten = (nodes: QuestionNode[]): QuestionNode[] => {
    let result: QuestionNode[] = [];
    nodes.forEach(n => {
      result.push(n);
      if (n.child_questions && n.child_questions.length > 0) {
        result = result.concat(tyrannicalFlatten(n.child_questions));
      }
    });
    return result;
  };
  const addRootQuestion = () => {
    const current = getCurrentQuestions();
    const newQuestion: QuestionNode = {
      id: crypto.randomUUID(),
      section: activeSubSection,
      sku_id: activeSubSection === 'sku' && skuTargetType === 'specific' ? skuTargetValue : null,
      sku_target_type: activeSubSection === 'sku' ? skuTargetType : 'all',
      sku_target_value: activeSubSection === 'sku' ? skuTargetValue : '',
      question_text: '',
      response_type: 'text',
      available_options: [],
      options_buffer: '',
      allow_gallery: false,      
      sort_order: current.length + 1,
      condition_branch: null,
      condition_operator: null,
      condition_value_text: [],
      condition_value_numeric: null,
      depends_on_question_id: null,
      depends_on_operator: null,
      depends_on_value: null,
      child_questions: []
    };
    setCurrentQuestions([...current, newQuestion]);
  };

  const addChildQuestion = (parentId: string) => {
    const current = getCurrentQuestions();
    const updateChildren = (list: QuestionNode[]): QuestionNode[] => {
      return list.map(q => {
        if (q.id === parentId) {
          const child: QuestionNode = {
            id: crypto.randomUUID(),
            section: q.section,
            sku_id: q.sku_id,
            sku_target_type: q.sku_target_type,
            sku_target_value: q.sku_target_value,
            question_text: '',
            response_type: 'text',
            available_options: [],
            options_buffer: '',
            allow_gallery: false,            
            sort_order: q.child_questions.length + 1,
            condition_branch: true,
            condition_operator: q.response_type === 'number' ? '<=' : 'equals',
            condition_value_text: [],
            condition_value_numeric: q.response_type === 'number' ? 0 : null,
            depends_on_question_id: null,
            depends_on_operator: null,
            depends_on_value: null,
            child_questions: []
          };
          return { ...q, child_questions: [...q.child_questions, child] };
        }
        return { ...q, child_questions: updateChildren(q.child_questions) };
      });
    };
    setCurrentQuestions(updateChildren(current));
  };

  const renderQuestionNode = (node: QuestionNode, index: number, totalAtLevel: number, depth = 0) => {
    const currentList = getCurrentQuestions();
    const allRootQuestions = currentList.filter(q => q.id !== node.id && !q.condition_branch);

    return (
      <div key={node.id} style={{ marginLeft: `${depth * 25}px`, padding: '15px', borderLeft: depth > 0 ? '2px dashed #0275d8' : '3px solid #ccc', marginBottom: '15px', backgroundColor: '#fdfdfd', borderRadius: '6px', border: '1px solid #eee' }}>
        <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', alignItems: 'center' }}>
          
          <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
            <button onClick={() => moveQuestion(node.id, 'up')} disabled={index === 0} style={{ padding: '2px 6px', fontSize: '10px' }}>▲</button>
            <button onClick={() => moveQuestion(node.id, 'down')} disabled={index === totalAtLevel - 1} style={{ padding: '2px 6px', fontSize: '10px' }}>▼</button>
          </div>
          
                    {depth === 0 && (
            <div style={{ display: 'flex', gap: '5px', alignItems: 'center', backgroundColor: '#eef7ff', padding: '6px', borderRadius: '4px', border: '1px solid #b3d7ff' }}>
              <span style={{ fontSize: '12px', fontWeight: 'bold', color: '#004085' }}>🔗 Зависит от вопроса:</span>
              <select
                value={node.depends_on_question_id || ''}
                onChange={e => {
                  const val = e.target.value;
                  // ИСПРАВЛЕНО: Безопасное сохранение без NaN. Если это числовой ID из БД — пишем число, если UUID — пишем строку.
                  const parsedId = val ? (isNaN(Number(val)) ? val : Number(val)) : null;
                  setCurrentQuestions(updateQuestionInTree(currentList, node.id, { 
                    depends_on_question_id: parsedId as any 
                  }));
                }}
                style={{ padding: '4px', fontSize: '12px' }}
              >
                <option value="">-- Нет зависимости --</option>
                {allRootQuestions.map((q, qIdx) => (
                  <option key={qIdx} value={q.db_id || q.id}>
                    {q.question_text || `Вопрос №${q.sort_order}`}
                  </option>
                ))}
              </select>

              {node.depends_on_question_id && (
                <div style={{ display: 'flex', gap: '3px', alignItems: 'center' }}>
                  <select
                    value={node.depends_on_operator || 'equals'}
                    onChange={e => setCurrentQuestions(updateQuestionInTree(currentList, node.id, { depends_on_operator: e.target.value }))}
                    style={{ padding: '4px', fontSize: '12px' }}
                  >
                    <option value="equals">=</option>
                    <option value=">">&gt;</option>
                    <option value="<">&lt;</option>
                    <option value=">=">&gt;=</option>
                    <option value="<=">&lt;=</option>
                    <option value="!=">!=</option>
                  </select>
                  <input 
                    type="text" 
                    placeholder="Значение" 
                    value={node.depends_on_value || ''}
                    onChange={e => setCurrentQuestions(updateQuestionInTree(currentList, node.id, { depends_on_value: e.target.value }))}
                    style={{ padding: '4px', width: '70px', fontSize: '12px' }}
                  />
                </div>
              )}
            </div>
          )}

          

          {depth > 0 && (
            <div style={{ display: 'flex', gap: '5px', alignItems: 'center', backgroundColor: '#eeb85c33', padding: '6px', borderRadius: '4px', border: '1px solid #ffeeba' }}>
              <span style={{ fontSize: '12px', fontWeight: 'bold' }}>Критерий ветки:</span>
              <select 
                value={node.condition_operator || 'equals'} 
                onChange={e => setCurrentQuestions(updateQuestionInTree(currentList, node.id, { condition_operator: e.target.value }))}
                style={{ padding: '4px', fontSize: '12px' }}
              >
                <option value="equals">Равно (=)</option>
                <option value="<=">&lt;=</option>
                <option value=">=">&gt;=</option>
                <option value="<">&lt;</option>
                <option value=">">&gt;</option>
              </select>
              <input 
                type="text" 
                placeholder="Значение" 
                value={node.condition_operator === 'equals' ? node.condition_value_text.join(' | ') : (node.condition_value_numeric ?? '')} 
                onChange={e => {
                  const val = e.target.value;
                  if (node.condition_operator === 'equals') {
                    setCurrentQuestions(updateQuestionInTree(currentList, node.id, { condition_value_text: [val] }));
                  } else {
                    setCurrentQuestions(updateQuestionInTree(currentList, node.id, { condition_value_numeric: val === '' ? null : Number(val) }));
                  }
                }}
                style={{ padding: '4px', width: '90px', fontSize: '12px' }}
              />
            </div>
          )}

          <input 
            type="text" 
            placeholder="Введите формулировку вопроса..." 
            value={node.question_text} 
            onChange={e => setCurrentQuestions(updateQuestionInTree(currentList, node.id, { question_text: e.target.value }))}
            style={{ flex: '1', minWidth: '200px', padding: '8px', border: '1px solid #ccc', borderRadius: '4px', fontSize: '14px' }}
          />

          <select 
            value={node.response_type} 
            onChange={e => setCurrentQuestions(updateQuestionInTree(currentList, node.id, { response_type: e.target.value as any }))}
            style={{ padding: '8px', borderRadius: '4px', border: '1px solid #ccc' }}
          >
            <option value="text">Тип: Текст</option>
            <option value="number">Тип: Число</option>
            <option value="date">Тип: Календарь (Дата)</option>
            <option value="photo">Тип: Фото выкладки</option>
            <option value="select">Тип: Список выбора</option>
            <option value="multiselect">Тип: Множественный выбор (Матч)</option> {/* ДОБАВЛЕНО */}
          </select>

          {node.response_type === 'photo' && (
            <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', backgroundColor: '#fff3cd', padding: '6px', borderRadius: '4px', border: '1px solid #ffeeba', cursor: 'pointer' }}>
              <input type="checkbox" checked={node.allow_gallery} onChange={e => setCurrentQuestions(updateQuestionInTree(currentList, node.id, { allow_gallery: e.target.checked }))} />
              <span style={{ fontWeight: 'bold', color: '#856404' }}>🔓 Разрешить Галерею</span>
            </label>
          )}
          
          {(node.response_type === 'select' || node.response_type === 'multiselect') && (
            <input 
              type="text" 
              placeholder="Опции через черту: Опция1 | Опция2 | Опция3" 
              value={node.options_buffer ?? ''}
              onChange={e => {
                const rawVal = e.target.value;
                const parsedArr = rawVal.split('|').map(s => s.trim()).filter(Boolean);
                setCurrentQuestions(updateQuestionInTree(currentList, node.id, { 
                  options_buffer: rawVal, 
                  available_options: parsedArr 
                }));
              }}
              style={{ padding: '8px', border: '1px solid #5cb85c', borderRadius: '4px', width: '250px', fontSize: '13px' }}
            />
          )}

          {(node.response_type === 'select' || node.response_type === 'number' || node.response_type === 'date') && (
            <button onClick={() => addChildQuestion(node.id)} style={{ padding: '8px 12px', backgroundColor: '#0275d8', color: '#fff', border: 'none', borderRadius: '4px', fontSize: '13px', cursor: 'pointer', fontWeight: 'bold' }}>+ Подвопрос</button>
          )}
          <button onClick={() => setCurrentQuestions(deleteQuestionFromTree(currentList, node.id))} style={{ padding: '8px 12px', backgroundColor: '#fff', color: '#d9534f', border: '1px solid #d9534f', borderRadius: '4px', cursor: 'pointer' }}>Удалить</button>
        </div>

        {node.child_questions.length > 0 && (
          <div style={{ marginTop: '15px' }}>
            {node.child_questions.map((child, cIdx) => renderQuestionNode(child, cIdx, node.child_questions.length, depth + 1))}
          </div>
        )}
      </div>
    );
  };
  return (
    <div style={{ padding: '10px 0' }}>
      <div style={{ display: 'flex', borderBottom: '2px solid #ccc', marginBottom: '20px' }}>
        <button onClick={() => setActiveSubSection('general')} style={{ padding: '10px 20px', cursor: 'pointer', border: 'none', borderBottom: activeSubSection === 'general' ? '4px solid #0275d8' : 'none', backgroundColor: 'transparent', fontWeight: 'bold', fontSize: '16px', color: activeSubSection === 'general' ? '#0275d8' : '#666' }}>
          1. ОБЩИЕ ВОПРОСЫ В ТОЧКЕ ({generalQuestions.length})
        </button>
        <button onClick={() => setActiveSubSection('sku')} style={{ padding: '10px 20px', cursor: 'pointer', border: 'none', borderBottom: activeSubSection === 'sku' ? '4px solid #0275d8' : 'none', backgroundColor: 'transparent', fontWeight: 'bold', fontSize: '16px', color: activeSubSection === 'sku' ? '#0275d8' : '#666' }}>
          2. ВОПРОСЫ ДЛЯ КАЖДОГО SKU ({skuQuestions.length})
        </button>
      </div>

      <div style={{ display: 'flex', gap: '20px', marginBottom: '20px', flexWrap: 'wrap', backgroundColor: '#f8f9fa', padding: '15px', borderRadius: '6px', border: '1px solid #eee' }}>
        <div>
          <label style={{ marginRight: '10px', fontSize: '14px', fontWeight: 'bold' }}>Производитель:</label>
          <select value={manufacturerId} onChange={e => setManufacturerId(e.target.value)} style={{ padding: '6px', borderRadius: '4px', border: '1px solid #ccc', fontWeight: 'bold' }}>
            <option value="">-- Выберите из базы --</option>
            {manufacturers.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
          </select>
        </div>

                {/* Интерактивный переключатель автоподстановки прошлых цен и остатков */}
        {manufacturerId && (
          <div style={{ marginLeft: '10px' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', backgroundColor: '#fff', padding: '6px 12px', borderRadius: '4px', border: '1px solid #ddd', cursor: 'pointer', fontWeight: 'bold', color: '#333' }}>
              <input 
                type="checkbox" 
                checked={isAutoCloneEnabled} 
                onChange={e => setIsAutoCloneEnabled(e.target.checked)} 
                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
              />
              📋 Включить автоподстановку прошлых ответов (плейсхолдеры на Pixel 4)
            </label>
          </div>
        )}

        {activeSubSection === 'sku' && (
          <>
            <div>
              <label style={{ marginRight: '10px', fontSize: '14px', fontWeight: 'bold' }}>Применить к:</label>
              <select value={skuTargetType} onChange={e => { setSkuTargetType(e.target.value as any); setSkuTargetValue(''); }} style={{ padding: '6px', borderRadius: '4px', border: '1px solid #ccc' }}>
                <option value="all">Всем SKU производителя</option>
                <option value="brand">Конкретному бренду</option>
                <option value="category">Конкретной категории</option>
                <option value="specific">Одному конкретному SKU</option>
              </select>
            </div>
            {skuTargetType !== 'all' && (
              <div>
                <label style={{ marginRight: '10px', fontSize: '14px', fontWeight: 'bold' }}>Выбор значения:</label>
                <select value={skuTargetValue} onChange={e => setSkuTargetValue(e.target.value)} style={{ padding: '6px', borderRadius: '4px', border: '1px solid #ccc' }}>
                  <option value="">-- Выберите из списка --</option>
                  {skuTargetType === 'specific' && dbSkus.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                  {skuTargetType === 'brand' && dbBrands.map(b => <option key={b} value={b}>{b}</option>)}
                  {skuTargetType === 'category' && dbCategories.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
            )}
          </>
        )}
      </div>

      {message && <div style={{ color: '#137333', backgroundColor: '#e6f4ea', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{message}</div>}
      {error && <div style={{ color: '#c5221f', backgroundColor: '#fce8e6', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{error}</div>}

      {manufacturerId ? (
        <div style={{ border: '1px solid #ddd', padding: '25px', borderRadius: '6px', backgroundColor: '#fafafa' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '25px' }}>
            <h4 style={{ margin: 0, color: '#333' }}>Конструирование: {activeSubSection === 'general' ? 'Общие вопросы в точке' : 'Вопросы для SKU'}</h4>
            <button onClick={addRootQuestion} style={{ padding: '8px 18px', backgroundColor: '#0275d8', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>+ Добавить вопрос верхнего уровня</button>
          </div>

          <div style={{ minHeight: '120px', marginBottom: '30px' }}>
            {getCurrentQuestions().length === 0 ? (
              <p style={{ color: '#999', fontStyle: 'italic', textAlign: 'center', paddingTop: '40px' }}>Раздел пуст. Нажмите кнопку выше, чтобы добавить первый вопрос.</p>
            ) : (
              getCurrentQuestions().map((node, idx) => renderQuestionNode(node, idx, getCurrentQuestions().length))
            )}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', borderTop: '2px solid #eee', paddingTop: '20px' }}>
            <button onClick={handleSmartSave} disabled={saving} style={{ padding: '14px 35px', backgroundColor: saving ? '#ccc' : '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold', fontSize: '16px', boxShadow: '0 2px 4px rgba(0,0,0,0.1)' }}>
              {saving ? 'Сохранение...' : 'Утвердить изменения анкеты'}
            </button>
          </div>
        </div>
      ) : (
        <p style={{ color: '#999', fontStyle: 'italic' }}>Выберите производителя из списка выше для активации конструктора.</p>
      )}
    </div>
  );
};
