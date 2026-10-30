package com.mg.sfa.ui.screens

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.mg.sfa.data.local.StoreEntity

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StoresScreen(
    storesList: List<StoreEntity>,
    activeVisitStoreId: String?,
    completedStoreIds: List<String>, // Динамический массив готовых точек из SQLite
    selectedStoreExternal: StoreEntity?,
    deviceLatitude: Double?,  // ДОБАВЛЕНО: Живая широта с датчика Pixel 4
    deviceLongitude: Double?,
    onStoreSelectedChange: (StoreEntity?) -> Unit,
    onStartVisitClick: (storeId: String) -> Unit
) {
    val selectedStore = selectedStoreExternal

    if (selectedStore == null) {
        Column(modifier = Modifier.fillMaxSize()) {
            Text(
                text = "Торговые точки",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 16.dp)
            )

            if (storesList.isEmpty()) {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Text("Список точек пуст.", color = MaterialTheme.colorScheme.outline)
                }
            } else {
 
                val context = androidx.compose.ui.platform.LocalContext.current
                val dao = remember { com.mg.sfa.data.local.AppDatabase.getDatabase(context).appDao() }
                
                var baseLat by remember { mutableStateOf<Double?>(deviceLatitude) }
                var baseLon by remember { mutableStateOf<Double?>(deviceLongitude) }

                LaunchedEffect(deviceLatitude, deviceLongitude, storesList) {
                    if (deviceLatitude != null && deviceLongitude != null) {
                        baseLat = deviceLatitude
                        baseLon = deviceLongitude
                    } else {
                        // Сценарий Подвал: Берем координаты последней успешно закрытой за сегодня ТТ
                        val lastClosedVisit = dao.getLastClosedVisitForStore("") // Пустая строка для получения глобально последнего визита
                        if (lastClosedVisit != null) {
                            val targetStore = storesList.find { it.id1c == lastClosedVisit.storeId1c }
                            if (targetStore != null && targetStore.latitude != 0.0) {
                                baseLat = targetStore.latitude
                                baseLon = targetStore.longitude
                            }
                        }
                    }
                }

               
                val sortedStores = remember(storesList, baseLat, baseLon) {
                    val currentLat = baseLat
                    val currentLon = baseLon
                    if (currentLat != null && currentLon != null) {
                        storesList.sortedBy { store ->
                            if (store.latitude == 0.0 && store.longitude == 0.0) Double.MAX_VALUE
                            else com.mg.sfa.util.SecurityUtils.calculateDistanceMeters(currentLat, currentLon, store.latitude, store.longitude)
                        }
                    } else {
                        storesList
                    }
                }


                                LazyColumn(
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                    modifier = Modifier.fillMaxSize()
                ) {
                    items(sortedStores) { store ->
                        val isCurrentActive = activeVisitStoreId == store.id1c
                        val isCompletedToday = completedStoreIds.contains(store.id1c)

                        val context = androidx.compose.ui.platform.LocalContext.current
                        val dao = remember { com.mg.sfa.data.local.AppDatabase.getDatabase(context).appDao() }
                        
                        var unsyncedBacklogCount by remember(store.id1c) { mutableStateOf(0) }
                        var answeredCountFromDb by remember(store.id1c, isCompletedToday) { mutableStateOf(0) }
                        var totalQuestionsCount by remember(store.id1c) { mutableStateOf(0) }

                                             
                        LaunchedEffect(store.id1c, isCompletedToday) {
                            unsyncedBacklogCount = dao.getUnsyncedVisitsCountForStore(store.id1c)
                            
                            val allQuestions = dao.getAllQuestions()
                            val storeManufacturers = dao.getManufacturersForStore(store.id1c)
                            var qSum = 0
                            storeManufacturers.forEach { m ->
                                qSum += allQuestions.count { it.manufacturerId == m.id && it.parentQuestionId == null }
                            }
                            totalQuestionsCount = qSum

                            if (isCompletedToday) {
                                val lastVisit = dao.getLastClosedVisitForStore(store.id1c)
                                answeredCountFromDb = lastVisit?.visitPercent ?: 0
                            }
                        }

                        val cardBgColor = when {
                            isCurrentActive -> Color(0xFFFFF7E6)
                            isCompletedToday -> Color(0xFFE6F4EA)
                            else -> MaterialTheme.colorScheme.surfaceVariant
                        }

                        val strokeColor = when {
                            isCurrentActive -> Color(0xFFFF9900)
                            isCompletedToday -> Color(0xFF137333)
                            else -> Color.Transparent
                        }

                        Card(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { onStoreSelectedChange(store) },
                            colors = CardDefaults.cardColors(containerColor = cardBgColor),
                            border = if (strokeColor != Color.Transparent) BorderStroke(1.5.dp, strokeColor) else null
                        ) {
                            Column(modifier = Modifier.padding(16.dp)) {
                                 Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Text(text = "${store.retailNetwork} — ${store.storeSign}", style = MaterialTheme.typography.titleSmall, modifier = Modifier.weight(1f))

                                    // ... Внутри Row разметки карточки, заменяем блок Column статусов справа:
                                    Column(
                                        horizontalAlignment = Alignment.End,
                                        verticalArrangement = Arrangement.spacedBy(2.dp)
                                    ) {
                                        when {
                                            isCurrentActive -> {
                                                Text(text = "● Визит свернут", color = Color(0xFFFF9900), fontWeight = FontWeight.Bold, fontSize = 12.sp)
                                            }
                                            isCompletedToday -> {
                                                Text(text = "✓ Завершен", color = Color(0xFF137333), fontWeight = FontWeight.Bold, fontSize = 12.sp)
                                                
                                                // ВЫВОДИМ СТРОГИЙ ФАКТ ВМЕСТО ПРОЦЕНТОВ
                                                Text(text = "Задач: $answeredCountFromDb/$totalQuestionsCount", fontSize = 11.sp, fontWeight = FontWeight.ExtraBold, color = MaterialTheme.colorScheme.primary)
                                                
                                                if (unsyncedBacklogCount > 0) {
                                                    Text(text = "⚠️ Надо отправить", fontSize = 10.sp, fontWeight = FontWeight.Black, color = Color.Red)
                                                }
                                            }
                                        }
                                    }
                                }

                                Text(text = store.address, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 4.dp))
                                Row(
                                    modifier = Modifier.fillMaxWidth().padding(top = 6.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                        Text(text = "ID: ${store.id1c}", fontSize = 11.sp, color = MaterialTheme.colorScheme.outline)
                                        if (baseLat != null && baseLon != null && store.latitude != 0.0) {
                                            val meters = com.mg.sfa.util.SecurityUtils.calculateDistanceMeters(baseLat!!, baseLon!!, store.latitude, store.longitude).toInt()
                                            val distanceText = if (meters >= 1000) "${String.format("%.1f", meters / 1000.0)} км" else "$meters м"
                                            Text(text = "📍 $distanceText", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.secondary)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }


            }
        }
      
        } else {
        val store = selectedStore
        val isAnotherVisitActive = activeVisitStoreId != null && activeVisitStoreId != store.id1c
        val isCompletedToday = completedStoreIds.contains(store.id1c)

        val context = androidx.compose.ui.platform.LocalContext.current
        val dao = remember { com.mg.sfa.data.local.AppDatabase.getDatabase(context).appDao() }

                
        var isAlreadySyncedByNetwork by remember(store.id1c) { mutableStateOf(false) }
        var isVisitReportEmpty by remember(store.id1c) { mutableStateOf(true) }

        
        
        var storeManufacturers by remember(store.id1c) { mutableStateOf<List<com.mg.sfa.data.local.ManufacturerEntity>>(emptyList()) }
        var allSkusList by remember { mutableStateOf<List<com.mg.sfa.data.local.SkuEntity>>(emptyList()) }
        var expandedManufacturerId by remember { mutableStateOf<Int?>(null) }

        
        LaunchedEffect(store.id1c) {
            storeManufacturers = dao.getManufacturersForStore(store.id1c)
            allSkusList = dao.getAllSkus()
            
            
            val lastClosedVisit = dao.getLastClosedVisitForStore(store.id1c)
            if (lastClosedVisit != null) {
                isAlreadySyncedByNetwork = lastClosedVisit.isSynced
                val answers = dao.getAnswersForVisit(lastClosedVisit.id)
                isVisitReportEmpty = answers.count { it.valueText.trim().isNotEmpty() } == 0
            } else {
                isAlreadySyncedByNetwork = false
                isVisitReportEmpty = true
            }
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(androidx.compose.foundation.rememberScrollState()),
            verticalArrangement = Arrangement.SpaceBetween
        ) {
            Column(modifier = Modifier.fillMaxWidth()) {
                Text(text = "${store.retailNetwork} — ${store.storeSign}", style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.primary)
                Text(text = "Формат точки: ${store.storeFormat}", style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(top = 4.dp))
                
                Divider(modifier = Modifier.padding(vertical = 12.dp))

                Text(text = "📍 Фактический адрес:", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                Text(text = store.address, style = MaterialTheme.typography.bodyLarge, modifier = Modifier.padding(top = 2.dp, bottom = 12.dp))

                Text(text = "👔 Ответственный Супервайзер:", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                Text(text = store.supervisorName, style = MaterialTheme.typography.bodyLarge, modifier = Modifier.padding(top = 2.dp))
                Text(text = "📞 Тел: ${store.supervisorPhone}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.outline, modifier = Modifier.padding(bottom = 12.dp))

                Divider(modifier = Modifier.padding(vertical = 8.dp))

               
                Text(
                    text = "📋 Производители:", 
                    style = MaterialTheme.typography.titleSmall, 
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.padding(bottom = 8.dp)
                )

                if (storeManufacturers.isEmpty()) {
                    Text(
                        text = "За данной торговой точкой не закреплено ни одного контракта производителя.", 
                        style = MaterialTheme.typography.bodyMedium, 
                        color = Color.Gray,
                        modifier = Modifier.padding(vertical = 8.dp)
                    )
                } else {
                    storeManufacturers.forEach { manufacturer ->
                        val isExpanded = expandedManufacturerId == manufacturer.id
                        
                        Card(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 4.dp)
                                .clickable { 
                                    expandedManufacturerId = if (isExpanded) null else manufacturer.id 
                                },
                            colors = CardDefaults.cardColors(
                                containerColor = if (isExpanded) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
                            )
                        ) {
                            Column(modifier = Modifier.padding(12.dp)) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Text(
                                        text = manufacturer.name, 
                                        style = MaterialTheme.typography.bodyLarge, 
                                        fontWeight = FontWeight.Bold,
                                        modifier = Modifier.weight(1f)
                                    )
                                    Text(
                                        text = if (isExpanded) "▼ Свернуть SKU" else "▶ Показать SKU", 
                                        style = MaterialTheme.typography.labelMedium,
                                        color = MaterialTheme.colorScheme.primary
                                    )
                                }
                                
                                Row(
                                    modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
                                    horizontalArrangement = Arrangement.spacedBy(16.dp)
                                ) {
                                    // Парсим числовой график дней недели в читаемый вид по маске ТЗ
                                    val daysMap = mapOf("1" to "Пн", "2" to "Вт", "3" to "Ср", "4" to "Чт", "5" to "Пт", "6" to "Сб", "7" to "Вс")
                                    val readableDays = manufacturer.workDaysStr.split(",")
                                        .mapNotNull { daysMap[it.trim()] }
                                        .joinToString(", ")

                                    Text(text = "⏱️ Время: ${manufacturer.durationMinutes} мин", style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Medium)
                                    Text(text = "📅 График: $readableDays", style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Medium)
                                }

                                // Если строка развернута — выводим номенклатуру SKU производителя
                                if (isExpanded) {
                                    Divider(modifier = Modifier.padding(vertical = 8.dp))
                                    val manufacturerSkus = allSkusList.filter { it.manufacturerId == manufacturer.id }
                                    
                                    if (manufacturerSkus.isEmpty()) {
                                        Text(text = "SKU для данного производителя отсутствуют в базе. Проведите синхронизацию.", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                                    } else {
                                        // Группируем SKU по категориям и брендам для чистого Data-Driven UI
                                        val groupedSkus = manufacturerSkus.groupBy { Pair(it.category, it.brand) }
                                        
                                        groupedSkus.forEach { (meta, skus) ->
                                            Text(
                                                text = "${meta.first.uppercase()} — БРЕНД: ${meta.second}", 
                                                fontSize = 11.sp, 
                                                fontWeight = FontWeight.ExtraBold, 
                                                color = MaterialTheme.colorScheme.secondary,
                                                modifier = Modifier.padding(top = 6.dp, bottom = 2.dp)
                                            )
                                            skus.forEach { sku ->
                                                Text(
                                                  text = "  • ${sku.name} (Код: ${sku.id})", 
                                                  style = MaterialTheme.typography.bodySmall,
                                                  modifier = Modifier.padding(vertical = 1.dp)
                                                )
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                
                if (isAnotherVisitActive) {
                    Card(
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                        modifier = Modifier.padding(top = 16.dp)
                    ) {
                        Text(
                            text = "⚠️ Внимание! У вас уже запущен активный визит в другой точке (ID: $activeVisitStoreId). Завершите его, прежде чем начать новый.",
                            color = MaterialTheme.colorScheme.onErrorContainer,
                            style = MaterialTheme.typography.bodyMedium,
                            modifier = Modifier.padding(12.dp)
                        )
                    }
                }
                
                if (isCompletedToday && !isAnotherVisitActive) {
                    Card(
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer),
                        modifier = Modifier.padding(top = 16.dp, bottom = 8.dp)
                    ) {
                        Text(
                            text = if (isVisitReportEmpty) {
                                "📦 Посещено сегодня: Только выкладка товаров (анкеты пусты). До отправки отчетов вы можете зайти внутрь и заполнить отчет."
                            } else {
                                "ℹ️ Вы уже завершили визит на этой точке. До синхронизации и отправки отчетов вы можете отредактировать любые внесенные данные."
                            },
                            color = MaterialTheme.colorScheme.onPrimaryContainer,
                            style = MaterialTheme.typography.bodyMedium, 
                            modifier = Modifier.padding(12.dp)
                        )
                    }
                }

            }

            Spacer(modifier = Modifier.height(24.dp))

            Row(
                modifier = Modifier.fillMaxWidth().padding(bottom = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                OutlinedButton(
                    onClick = { onStoreSelectedChange(null) },
                    modifier = Modifier.weight(1f)
                ) {
                    Text("Назад")
                }               

                                
                var isVisitExpired by remember(store.id1c) { mutableStateOf(false) }
                
                LaunchedEffect(store.id1c, activeVisitStoreId) {
                    val activeVisit = dao.getActiveVisit()
                    if (activeVisit != null && activeVisit.storeId1c == store.id1c) {
                        val timeFormatter = java.text.SimpleDateFormat("yyyy-MM-dd HH:mm", java.util.Locale.US)
                        try {
                            val safeStartedAt = activeVisit.startedAt.replace("T", " ").let { if (it.length > 16) it.substring(0, 16) else it }
                            val startedDate = timeFormatter.parse(safeStartedAt)
                            if (startedDate != null) {
                                val diffMs = java.util.Calendar.getInstance().time.time - startedDate.time
                                isVisitExpired = (diffMs / (1000 * 60 * 60)) >= 24
                            }
                        } catch (e: Exception) {}
                    } else {
                        isVisitExpired = false
                    }
                }

                Button(
                    onClick = {
                        onStartVisitClick(store.id1c)
                        onStoreSelectedChange(null)
                    },
                    enabled = !isAnotherVisitActive,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = if (isVisitExpired) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary
                    ),
                    modifier = Modifier.weight(1.5f)
                ) {
                    Text(
                        text = when {
                            isVisitExpired -> "⚠️ Истекло 24 часа. Закрыть"
                            activeVisitStoreId == store.id1c -> "Продолжить визит"
                            isCompletedToday && !isAlreadySyncedByNetwork -> "Редактировать отчет"
                            isCompletedToday && isAlreadySyncedByNetwork -> "Повторный визит"
                            else -> "Начать визит"
                        }
                    )
                }


            }
        }
    }
}
