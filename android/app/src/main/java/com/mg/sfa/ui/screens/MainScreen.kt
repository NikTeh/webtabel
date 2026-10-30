package com.mg.sfa.ui.screens
import com.mg.sfa.data.local.ManufacturerEntity
import com.mg.sfa.data.local.StoreManufacturerLinkEntity

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.mg.sfa.data.local.StoreEntity

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(
    manufacturersList: List<ManufacturerEntity>,          // ИСПРАВЛЕНО: Принимаем массив А
    storeManufacturerLinks: List<StoreManufacturerLinkEntity>, // ИСПРАВЛЕНО: Принимаем массив Б
    meName: String,
    storesList: List<StoreEntity>,
    activeVisitStoreId: String?,        
    unsyncedCount: Int,
    completedStoreIds: List<String>,
    // completedStorePercents: Map<String, Int> = emptyMap(), 
    initialTabInternal: Int,
    deviceLatitude: Double?,  // ДОБАВЛЕНО ДЛЯ ПОД КЛЮЧ ГЕО
    deviceLongitude: Double?,         
    onStartVisitClick: (storeId: String, sourceTab: Int) -> Unit,
    // ИСПРАВЛЕНО: Добавлены типы Callback-слушателей для честного онлайн-статуса
    onUploadReportsClick: (onResult: (Int) -> Unit) -> Unit,
    onSyncDatabaseClick: (onResult: (Boolean) -> Unit) -> Unit,
    onLogout: () -> Unit
) {
    var selectedTab by remember(initialTabInternal) { mutableStateOf(initialTabInternal) }
    var selectedStoreForCard by remember { mutableStateOf<StoreEntity?>(null) }
    var navigationSource by remember { mutableStateOf(if (initialTabInternal == 1) "TRACK_ROUTES" else "TRACK_POINTS") }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("$meName") },
                actions = {
                    IconButton(onClick = onLogout) { Text("🚪") }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primary,
                    titleContentColor = MaterialTheme.colorScheme.onPrimary
                )
            )
        },
        bottomBar = {
            NavigationBar {
                NavigationBarItem(
                    selected = selectedTab == 0,
                    onClick = { 
                        selectedTab = 0 
                        selectedStoreForCard = null
                        navigationSource = "TRACK_POINTS"
                    },
                    label = { Text("Точки") },
                    icon = { Text("🏪", style = MaterialTheme.typography.titleLarge) }
                )
                NavigationBarItem(
                    selected = selectedTab == 1,
                    onClick = { 
                        selectedTab = 1 
                        selectedStoreForCard = null
                        navigationSource = "TRACK_ROUTES"
                    },
                    label = { Text("Маршруты") },
                    icon = { Text("📅", style = MaterialTheme.typography.titleLarge) }
                )
                NavigationBarItem(
                    selected = selectedTab == 2,
                    onClick = { selectedTab = 2 },
                    label = { Text("Настройки") },
                    icon = { 
                        BadgedBox(badge = {
                            if (unsyncedCount > 0) {
                                Badge { Text("$unsyncedCount") }
                            }
                        }) {
                            Text("⚙️", style = MaterialTheme.typography.titleLarge)
                        }
                    }
                )
            }
        }
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp)
        ) {
            if (selectedStoreForCard != null) {
                StoresScreen(
                    storesList = storesList,
                    activeVisitStoreId = activeVisitStoreId,
                    completedStoreIds = completedStoreIds,
                    selectedStoreExternal = selectedStoreForCard,
                    deviceLatitude = deviceLatitude,   // ПРОБРОС ШИРОТЫ В КАРТОЧКУ
                    deviceLongitude = deviceLongitude,
                    onStoreSelectedChange = { store ->
                        selectedStoreForCard = store
                        if (store == null) {
                            selectedTab = if (navigationSource == "TRACK_ROUTES") 1 else 0
                        }
                    },
                    onStartVisitClick = { id -> onStartVisitClick(id, selectedTab) }
                )
            } else {
                when (selectedTab) {
                    0 -> {
                        StoresScreen(
                            storesList = storesList,
                            activeVisitStoreId = activeVisitStoreId,
                            completedStoreIds = completedStoreIds,
                            selectedStoreExternal = null,
                            deviceLatitude = deviceLatitude,   // ПРОБРОС ШИРОТЫ
                            deviceLongitude = deviceLongitude,
                            onStoreSelectedChange = { 
                                selectedStoreForCard = it 
                                navigationSource = "TRACK_POINTS"
                            },
                            onStartVisitClick = { id -> onStartVisitClick(id, 0) }
                        )
                    }
                    
                                        1 -> {
                        RoutesScreen(
                            allStores = storesList,
                            manufacturersList = manufacturersList,          // ИСПРАВЛЕНО: Прокидываем оффлайн-карточки брендов
                            storeManufacturerLinks = storeManufacturerLinks, // ИСПРАВЛЕНО: Прокидываем плоские связи контрактов
                            activeVisitStoreId = activeVisitStoreId,
                            completedStoreIds = completedStoreIds,
                            // completedStorePercents = completedStorePercents, 
                            deviceLatitude = deviceLatitude,   // ПРОБРОС ШИРОТЫ В МАРШРУТЫ
                            deviceLongitude = deviceLongitude, 
                            onStoreClick = { storeId ->
                                val targetStore = storesList.find { it.id1c == storeId }
                                if (targetStore != null) {
                                    navigationSource = "TRACK_ROUTES"
                                    selectedStoreForCard = targetStore
                                }
                            }
                        )
                    }

                    2 -> {
                        SettingsScreen(
                            unsyncedReportsCount = unsyncedCount,
                            
                            onSyncDatabaseClick = onSyncDatabaseClick,
                            onUploadReportsClick = onUploadReportsClick
                        )
                    }
                }
            }
        }
    }
}
