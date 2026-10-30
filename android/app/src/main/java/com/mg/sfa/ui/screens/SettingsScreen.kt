package com.mg.sfa.ui.screens

import androidx.compose.ui.graphics.Color
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    unsyncedReportsCount: Int, // Реальное количество неотправленных отчетов из БД SQLite
    onSyncDatabaseClick: (onResult: (Boolean) -> Unit) -> Unit, // Проводник с обратным вызовом (Callback)
    onUploadReportsClick: (onResult: (Int) -> Unit) -> Unit     // Проводник с обратным вызовом (Callback)
) {
    var isSyncingData by remember { mutableStateOf(false) }
    var isUploadingReports by remember { mutableStateOf(false) }
    
    // Системный менеджер для всплывающих окон Jetpack Compose
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    Scaffold(
        // ИСПРАВЛЕНО: Закрепляем системный хост уведомлений строго внизу экрана Scaffold
        snackbarHost = { SnackbarHost(hostState = snackbarHostState) },
        containerColor = Color.Transparent // Позволяет сохранить общий фон приложения
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(8.dp)
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = "Системные настройки и синхронизация",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(bottom = 8.dp)
            )

            
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(text = "Синхронизация данных", fontWeight = FontWeight.Bold, fontSize = 15.sp)
                    Text(
                        text = "Скачивание актуальной адресной программы, SKU-листов, контрактов и свежих версий анкет производителей.",
                        fontSize = 12.sp,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(vertical = 8.dp)
                    )
                    Button(
                        onClick = {
                            isSyncingData = true
                            // Запускаем корутину и ждем честный ответ от сервера
                            onSyncDatabaseClick { isSuccess ->
                                isSyncingData = false
                                scope.launch {
                                    if (isSuccess) {
                                        snackbarHostState.showSnackbar("✅ Данные обновлены.")
                                    } else {
                                        snackbarHostState.showSnackbar("❌ Ошибка! Проверьте подключение к интернету.")
                                    }
                                }
                            }
                        },
                        enabled = !isSyncingData,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text(if (isSyncingData) "Синхронизация..." else "Синхронизировать данные")
                    }
                }
            }

            // --- БЛОК 2: ВЫГРУЗКА ПоЛЕВЫХ ОТЧЕТОВ ---
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(text = "Отправка отчетов", fontWeight = FontWeight.Bold, fontSize = 15.sp)
                        
                        if (unsyncedReportsCount > 0) {
                            Badge(containerColor = MaterialTheme.colorScheme.error) {
                                Text(
                                    text = " $unsyncedReportsCount ",
                                    color = MaterialTheme.colorScheme.onError,
                                    fontWeight = FontWeight.Bold,
                                    modifier = Modifier.padding(4.dp)
                                )
                            }
                        }
                    }
                    
                    Text(
                        text = "Отправка накопленных в оффлайне визитов, ответов и фотографий выкладки.",
                        fontSize = 12.sp,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(vertical = 8.dp)
                    )
                    
                    Button(
                        onClick = {
                            if (unsyncedReportsCount == 0) {
                                scope.launch { snackbarHostState.showSnackbar("ℹ️ Нет неотправленных отчетов.") }
                                return@Button
                            }
                            isUploadingReports = true
                            // Передаем лямбду-слушатель, чтобы узнать точное число синхронизированных визитов
                            onUploadReportsClick { syncedCount ->
                                isUploadingReports = false
                                scope.launch {
                                    if (syncedCount > 0) {
                                        snackbarHostState.showSnackbar("🚀 Успешно отправлено отчетов: $syncedCount")
                                    } else {
                                        snackbarHostState.showSnackbar("❌ Сбой отправки отчетов. Проверьте сеть или авторизацию.")
                                    }
                                }
                            }
                        },
                        enabled = !isUploadingReports,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = if (unsyncedReportsCount > 0) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary
                        ),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text(if (isUploadingReports) "Отправка..." else "Отправить отчёты")
                    }
                }
            }

            // --- БЛОК 3: ТЕХНИЧЕСКАЯ ПОДДЕРЖКА ---
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(text = "💬 Служба технической поддержки", fontWeight = FontWeight.Bold, fontSize = 15.sp)
                    Text(
                        text = "Если у вас возникли проблемы с оффлайн-доступом или отправкой фотографий выкладки товаров, напишите нашему боту:",
                        fontSize = 12.sp,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp, bottom = 12.dp)
                    )
                    
                    val context = androidx.compose.ui.platform.LocalContext.current
                    Button(
                        onClick = {
                            val intent = android.content.Intent(
                                android.content.Intent.ACTION_VIEW,
                                android.net.Uri.parse("https://t.me")
                            )
                            context.startActivity(intent)
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF2481CC)),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Написать в Telegram: @WebtabelBot", fontSize = 13.sp, fontWeight = FontWeight.Bold)
                    }
                }
            }

            Spacer(modifier = Modifier.height(60.dp))
        }
    }
}
