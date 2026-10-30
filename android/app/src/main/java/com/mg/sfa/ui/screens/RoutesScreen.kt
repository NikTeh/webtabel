package com.mg.sfa.ui.screens

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
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
import com.mg.sfa.data.local.ManufacturerEntity
import com.mg.sfa.data.local.StoreManufacturerLinkEntity

data class RouteStoreItem(
    val store: StoreEntity,
    val manufacturerCount: Int,
    val totalDurationMinutes: Int
)

@Composable
fun RoutesScreen(
    allStores: List<StoreEntity>,
    manufacturersList: List<ManufacturerEntity>, 
    storeManufacturerLinks: List<StoreManufacturerLinkEntity>, 
    activeVisitStoreId: String?,
    completedStoreIds: List<String>,   
    deviceLatitude: Double?,  
    deviceLongitude: Double?, 
    onStoreClick: (storeId: String) -> Unit
) {
    var selectedDayOfWeek by remember { mutableStateOf(1) }


    val daysOfWeekLabels = listOf(
        1 to "Пн", 2 to "Вт", 3 to "Ср", 4 to "Чт", 5 to "Пт", 6 to "Сб", 7 to "Вс"
    )

        
    val context = androidx.compose.ui.platform.LocalContext.current
    val dao = remember { com.mg.sfa.data.local.AppDatabase.getDatabase(context).appDao() }
    
    var baseLat by remember { mutableStateOf<Double?>(deviceLatitude) }
    var baseLon by remember { mutableStateOf<Double?>(deviceLongitude) }

    LaunchedEffect(deviceLatitude, deviceLongitude, allStores) {
        if (deviceLatitude != null && deviceLongitude != null) {
            baseLat = deviceLatitude
            baseLon = deviceLongitude
        } else {
            
            val lastClosedVisit = dao.getLastClosedVisitForStore("")
            if (lastClosedVisit != null) {
                val targetStore = allStores.find { it.id1c == lastClosedVisit.storeId1c }
                if (targetStore != null && targetStore.latitude != 0.0) {
                    baseLat = targetStore.latitude
                    baseLon = targetStore.longitude
                }
            }
        }
    }

        
    val routeItems = remember(selectedDayOfWeek, allStores, manufacturersList, storeManufacturerLinks, baseLat, baseLon) {
        if (allStores.isEmpty() || manufacturersList.isEmpty() || storeManufacturerLinks.isEmpty()) {
            emptyList()
        } else {
            val activeManufacturersForToday = manufacturersList.filter { man ->
                man.workDaysStr.split(",").map { it.trim() }.contains(selectedDayOfWeek.toString())
            }
            val activeManIds = activeManufacturersForToday.map { it.id }

            val filteredList = allStores.mapNotNull { store ->
                val storeLinks = storeManufacturerLinks.filter { it.storeId1c == store.id1c }
                val todayStoreBrands = storeLinks.filter { it.manufacturerId in activeManIds }

                if (todayStoreBrands.isEmpty()) {
                    null
                } else {
                    val manufacturersCount = todayStoreBrands.size
                    val totalDuration = activeManufacturersForToday
                        .filter { it.id in todayStoreBrands.map { link -> link.manufacturerId } }
                        .sumOf { it.durationMinutes }

                    RouteStoreItem(
                        store = store,
                        manufacturerCount = manufacturersCount,
                        totalDurationMinutes = totalDuration
                    )
                }
            }

            
            val currentLat = baseLat
            val currentLon = baseLon
            if (currentLat != null && currentLon != null) {
                filteredList.sortedBy { item ->
                    if (item.store.latitude == 0.0 && item.store.longitude == 0.0) Double.MAX_VALUE
                    else com.mg.sfa.util.SecurityUtils.calculateDistanceMeters(currentLat, currentLon, item.store.latitude, item.store.longitude)
                }
            } else {
                filteredList
            }
        }
    }
   

        Column(modifier = Modifier.fillMaxSize()) {
            Text(
                text = "Маршруты по дням недели",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 12.dp)
            )

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                daysOfWeekLabels.forEach { (dayValue, label) ->
                    val isSelected = selectedDayOfWeek == dayValue
                    Button(
                        onClick = { selectedDayOfWeek = dayValue },
                        colors = ButtonDefaults.buttonColors(
                            containerColor = if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant,
                            contentColor = if (isSelected) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant
                        ),
                        contentPadding = PaddingValues(0.dp),
                        modifier = Modifier
                            .weight(1f)
                            .height(40.dp)
                    ) {
                        Text(text = label, fontWeight = FontWeight.Bold, fontSize = 13.sp)
                    }
                }
            }

            if (routeItems.isEmpty()) {
                Box(modifier = Modifier.fillMaxSize().weight(1f), contentAlignment = Alignment.Center) {
                    Text(
                        text = "На этот день плановые визиты отсутствуют",
                        color = MaterialTheme.colorScheme.outline,
                        fontSize = 14.sp
                    )
                }
            } else {

                LazyColumn(
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                    modifier = Modifier.fillMaxWidth().weight(1f)
                ) {
                    items(routeItems) { item ->
                        
                        val isCurrentActive = activeVisitStoreId == item.store.id1c
                        val isCompletedToday = completedStoreIds.contains(item.store.id1c)

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

                        

                        val context = androidx.compose.ui.platform.LocalContext.current
                        val dao = remember { com.mg.sfa.data.local.AppDatabase.getDatabase(context).appDao() }
                        
                        var routeBacklogCount by remember(item.store.id1c) { mutableStateOf(0) }
                        var answeredCountFromDb by remember(item.store.id1c, isCompletedToday) { mutableStateOf(0) }
                        var totalQuestionsCount by remember(item.store.id1c) { mutableStateOf(0) }

                        LaunchedEffect(item.store.id1c, isCompletedToday) {
                            routeBacklogCount = dao.getUnsyncedVisitsCountForStore(item.store.id1c)
                            
                            
                            val allQuestions = dao.getAllQuestions()
                            val storeManufacturers = dao.getManufacturersForStore(item.store.id1c)
                            var qSum = 0
                            storeManufacturers.forEach { m ->
                                qSum += allQuestions.count { it.manufacturerId == m.id && it.parentQuestionId == null }
                            }
                            totalQuestionsCount = qSum

                            if (isCompletedToday) {
                                val lastVisit = dao.getLastClosedVisitForStore(item.store.id1c)
                                answeredCountFromDb = lastVisit?.visitPercent ?: 0
                            }
                        }

                        Card(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { onStoreClick(item.store.id1c) },
                            colors = CardDefaults.cardColors(containerColor = cardBgColor),
                            border = if (strokeColor != Color.Transparent) BorderStroke(1.5.dp, strokeColor) else null
                        ) {
                            Column(modifier = Modifier.padding(16.dp)) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Text(
                                        text = "${item.store.retailNetwork} — ${item.store.storeSign}",
                                        style = MaterialTheme.typography.titleSmall,
                                        modifier = Modifier.weight(1f)
                                    )

                                    Row(
                                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Surface(
                                            color = MaterialTheme.colorScheme.primaryContainer,
                                            shape = MaterialTheme.shapes.small
                                        ) {
                                            Text(
                                                text = "⏱️ ${item.totalDurationMinutes} мин",
                                                style = MaterialTheme.typography.labelMedium,
                                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 3.dp)
                                            )
                                        }
                                        
                                       
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
                                                    
                                                    
                                                    Text(
                                                        text = "Задач: $answeredCountFromDb/$totalQuestionsCount", 
                                                        fontSize = 11.sp, 
                                                        fontWeight = FontWeight.ExtraBold, 
                                                        color = MaterialTheme.colorScheme.primary
                                                    )
                                                    
                                                    if (routeBacklogCount > 0) {
                                                        Text(text = "⚠️ Надо отправить", fontSize = 10.sp, fontWeight = FontWeight.Black, color = Color.Red)
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }


                               Text(
                                    text = item.store.address,
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    modifier = Modifier.padding(top = 4.dp)
                                )
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(top = 6.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {                                   
                                    // РАСШИРЕННЫЙ ВЕРТИКАЛЬНЫЙ ПОДВАЛ ДЛЯ ТТ
                                    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                        Text(text = "ID: ${item.store.id1c}", fontSize = 11.sp, color = MaterialTheme.colorScheme.outline)
                                        
                                        // Перенесено вниз слева под 1С ID точки: экономит место на Pixel 4
                                        if (baseLat != null && baseLon != null && item.store.latitude != 0.0) {
                                            val meters = com.mg.sfa.util.SecurityUtils.calculateDistanceMeters(baseLat!!, baseLon!!, item.store.latitude, item.store.longitude).toInt()
                                            val distanceText = if (meters >= 1000) "${String.format("%.1f", meters / 1000.0)} км" else "$meters м"
                                            Text(text = "📍 $distanceText", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.secondary)
                                        }
                                    }

                                    Text(
                                        text = "Производителей: ${item.manufacturerCount}",
                                        fontSize = 12.sp,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.primary
                                    )
                                }


                            }
                        }
                    }
                  } 
        }
    }
}
