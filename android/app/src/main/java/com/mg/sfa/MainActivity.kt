package com.mg.sfa


import androidx.compose.foundation.background
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.ui.unit.dp


import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier


import androidx.lifecycle.lifecycleScope

import com.mg.sfa.data.local.AppDatabase
import com.mg.sfa.data.local.StoreEntity
import com.mg.sfa.data.local.VisitEntity
import com.mg.sfa.data.local.AnswerEntity
import com.mg.sfa.data.local.QuestionEntity

import com.mg.sfa.data.local.ManufacturerEntity
import com.mg.sfa.data.local.StoreManufacturerLinkEntity

import com.mg.sfa.data.remote.ApiService
import com.mg.sfa.data.remote.*
import com.mg.sfa.data.repository.DataRepository
import com.mg.sfa.ui.screens.LoginScreen
import com.mg.sfa.ui.screens.MainScreen
import com.mg.sfa.ui.screens.VisitScreen
import kotlinx.coroutines.launch
import java.util.UUID

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val database = AppDatabase.getDatabase(applicationContext)
        val dao = database.appDao()

        setContent {
            MaterialTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    
                    val permissionLauncher = androidx.activity.compose.rememberLauncherForActivityResult(
                        contract = androidx.activity.result.contract.ActivityResultContracts.RequestMultiplePermissions()
                    ) { permissions ->
                        val fineGranted = permissions[android.Manifest.permission.ACCESS_FINE_LOCATION] ?: false
                        val coarseGranted = permissions[android.Manifest.permission.ACCESS_COARSE_LOCATION] ?: false
                        val cameraGranted = permissions[android.Manifest.permission.CAMERA] ?: false
                        
                        if (!fineGranted && !coarseGranted) {
                            android.widget.Toast.makeText(
                                applicationContext,
                                "⚠️ Внимание! Без доступа к GPS координаты визитов будут заблокированы.",
                                android.widget.Toast.LENGTH_LONG
                            ).show()
                        }
                        if (!cameraGranted) {
                            android.widget.Toast.makeText(
                                applicationContext,
                                "⚠️ Внимание! Без доступа к КАМЕРЕ фотоотчеты будут недоступны.",
                                android.widget.Toast.LENGTH_LONG
                            ).show()
                        }
                    }

                    var isAuthenticated by remember { mutableStateOf(false) }
                    var meName by remember { mutableStateOf("") }
                    var mePhone by remember { mutableStateOf("") }
                    var mePassword by remember { mutableStateOf("") }
                    
                    var storesList by remember { mutableStateOf<List<StoreEntity>>(emptyList()) }
                    var activeVisitStoreId by remember { mutableStateOf<String?>(null) }
                    var activeVisitId by remember { mutableStateOf<String?>(null) }
                    var savedAnswers by remember { mutableStateOf<List<AnswerEntity>>(emptyList()) }
                    var questionsList by remember { mutableStateOf<List<QuestionEntity>>(emptyList()) }
                    
                    var manufacturersList by remember { mutableStateOf<List<ManufacturerEntity>>(emptyList()) }
                    var storeManufacturerLinks by remember { mutableStateOf<List<StoreManufacturerLinkEntity>>(emptyList()) }

                    var deviceLatitude by remember { mutableStateOf<Double?>(null) }  
                    var deviceLongitude by remember { mutableStateOf<Double?>(null) } 


                    var unsyncedCount by remember { mutableStateOf(0) }

                    var currentScreen by remember { mutableStateOf("LOGIN") }
                    var visitSubscreenSource by remember { mutableStateOf(0) }
                    var completedStoreIds by remember { mutableStateOf<List<String>>(emptyList()) }

                    var testJsonDiagnosticData by remember { mutableStateOf<String?>(null) }

                   LaunchedEffect(isAuthenticated, currentScreen) {
                        if (isAuthenticated) {
                            permissionLauncher.launch(
                                arrayOf(
                                    android.Manifest.permission.ACCESS_FINE_LOCATION,
                                    android.Manifest.permission.ACCESS_COARSE_LOCATION,
                                    android.Manifest.permission.CAMERA
                                )
                            )

                            storesList = dao.getAllStores()
                            questionsList = dao.getAllQuestions()
                            manufacturersList = dao.getAllManufacturers()
                            storeManufacturerLinks = dao.getAllStoreManufacturerLinks()

                            try {
                                val locManager = applicationContext.getSystemService(android.content.Context.LOCATION_SERVICE) as android.location.LocationManager
                                val providers = locManager.getProviders(true)
                                var bestLocation: android.location.Location? = null
                                for (provider in providers) {
                                    val l = locManager.getLastKnownLocation(provider) ?: continue
                                    if (bestLocation == null || l.accuracy < bestLocation.accuracy) {
                                        bestLocation = l
                                    }
                                }
                                if (bestLocation != null) {
                                    deviceLatitude = bestLocation.latitude
                                    deviceLongitude = bestLocation.longitude
                                }
                            } catch (e: SecurityException) {
                               
                            }

                            var activeVisit = dao.getActiveVisit()
                            if (activeVisit != null) {
                                val timeFormatter = java.text.SimpleDateFormat("yyyy-MM-dd HH:mm", java.util.Locale.US)
                                try {
                                    val safeStartedAt = activeVisit.startedAt
                                        .replace("T", " ")
                                        .let { if (it.length > 16) it.substring(0, 16) else it }

                                    val startedDate = timeFormatter.parse(safeStartedAt)
                                    val currentDate = java.util.Calendar.getInstance().time
                                    
                                    if (startedDate != null) {
                                        val diffMs = currentDate.time - startedDate.time
                                        val diffHours = diffMs / (1000 * 60 * 60)
                                        
                                        if (diffHours >= 24) {
                                            val cal = java.util.Calendar.getInstance().apply { time = startedDate }
                                            cal.add(java.util.Calendar.HOUR_OF_DAY, 24)
                                            val targetEndedAt = timeFormatter.format(cal.time)

                                            dao.saveVisit(activeVisit.copy(endedAt = targetEndedAt, isSynced = false))
                                            activeVisit = null 
                                            
                                            android.widget.Toast.makeText(
                                                applicationContext,
                                                "ℹ️ Предыдущий открытый визит автоматически завершен (истек лимит 24 часа).",
                                                android.widget.Toast.LENGTH_LONG
                                            ).show()
                                        }
                                    }
                                } catch (e: Exception) {}
                            }

                            if (currentScreen != "VISIT") {
                                activeVisitStoreId = activeVisit?.storeId1c
                                activeVisitId = activeVisit?.id
                                if (activeVisit != null) {
                                    savedAnswers = dao.getAnswersForVisit(activeVisit.id)
                                } else {
                                    savedAnswers = emptyList()
                                }
                            }
                            
                            val unsyncedVisits = dao.getUnsyncedVisits()
                            unsyncedCount = unsyncedVisits.size
                            
                            val todayPrefix = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.US).format(java.util.Calendar.getInstance().time)
                            completedStoreIds = dao.getVisitedStoreIdsForToday(todayPrefix)
                        }
                    }

                    


                    val onAnswerChangeLambda: (String, String, String, Double?) -> Unit = { qId, sku, text, num ->
                        val currentVisitId = activeVisitId
                        if (currentVisitId != null) {
                            val newAnswer = AnswerEntity(
                                visitId = currentVisitId,
                                questionId = qId,
                                skuId = sku,
                                valueText = text,
                                valueNumber = num
                            )
                            lifecycleScope.launch {
                                dao.saveAnswer(newAnswer)
                                savedAnswers = dao.getAnswersForVisit(currentVisitId)
                            }
                        }
                    }
                   
                                       
                    val fastOkHttpClient = remember {
                        okhttp3.OkHttpClient.Builder()
                            .connectTimeout(3, java.util.concurrent.TimeUnit.SECONDS) 
                            .readTimeout(3, java.util.concurrent.TimeUnit.SECONDS)
                            .writeTimeout(3, java.util.concurrent.TimeUnit.SECONDS)
                            .build()
                    }

                    val heavyOkHttpClient = remember {
                        okhttp3.OkHttpClient.Builder()
                            .connectTimeout(30, java.util.concurrent.TimeUnit.SECONDS)
                            .readTimeout(60, java.util.concurrent.TimeUnit.SECONDS)
                            .writeTimeout(90, java.util.concurrent.TimeUnit.SECONDS) 
                            .build()
                    }
                   
                    val retrofitFast = remember {
                        retrofit2.Retrofit.Builder()                            
                            .baseUrl("https://webtabel.ru")
                            .client(fastOkHttpClient) 
                            .addConverterFactory(retrofit2.converter.gson.GsonConverterFactory.create())
                            .build()
                    }

                    val retrofitHeavy = remember {
                        retrofit2.Retrofit.Builder()                            
                            .baseUrl("https://webtabel.ru")
                            .client(heavyOkHttpClient) 
                            .addConverterFactory(retrofit2.converter.gson.GsonConverterFactory.create())
                            .build()
                    }

                    val apiService = remember { retrofitFast.create(ApiService::class.java) }
                    val apiServiceHeavy = remember { retrofitHeavy.create(ApiService::class.java) }
                    val dataRepository = remember { DataRepository(dao, apiServiceHeavy) } 

                    when (currentScreen) {
                        "LOGIN" -> {
                            LoginScreen(
                                apiService = apiService,
                                dao = dao,
                                onLoginSuccess = { phone, pass, name ->
                                    mePhone = phone
                                    mePassword = pass
                                    meName = name
                                    isAuthenticated = true
                                    currentScreen = "MAIN"
                                }
                            )
                        }

                        
                        "VISIT" -> {
                            val currentActiveStoreId = activeVisitStoreId
                            if (currentActiveStoreId != null) {
                                val activeStoreObj = storesList.find { it.id1c == currentActiveStoreId }
                                val addressStr = activeStoreObj?.address ?: "Адрес..."
                                val storeSignStr = activeStoreObj?.storeSign ?: "Магазин"
                                
                                var localSkusList by remember { mutableStateOf<List<com.mg.sfa.data.local.SkuEntity>>(emptyList()) }
                                LaunchedEffect(currentActiveStoreId) {
                                    localSkusList = dao.getAllSkus()
                                }
                                
                                VisitScreen(
                                    storeId1c = currentActiveStoreId,
                                    storeSign = storeSignStr, 
                                    storeAddress = addressStr,
                                    questionsList = questionsList,
                                    skusList = localSkusList, 
                                    manufacturersList = manufacturersList,
                                    storeManufacturerLinks = storeManufacturerLinks,
                                    savedAnswers = savedAnswers,
                                    onAnswerChange = onAnswerChangeLambda,

                                    onBackToMainMenuClick = {
                                        currentScreen = "MAIN"
                                    },

                                    onCloseVisitClick = {
                                        lifecycleScope.launch {
                                            val activeVisit = dao.getActiveVisit()
                                            if (activeVisit != null) {
                                                val timeFormatter = java.text.SimpleDateFormat("yyyy-MM-dd HH:mm", java.util.Locale.US)
                                                val liveEndedAt = timeFormatter.format(java.util.Calendar.getInstance().time)

                                                // Считаем чистый факт введенных ответов Николая по всей точке
                                                val answers = dao.getAnswersForVisit(activeVisit.id)
                                                val rawAnsweredCount = answers.count { it.valueText.trim().isNotEmpty() }

                                                val closedVisit = activeVisit.copy(
                                                    endedAt = liveEndedAt,
                                                    isSynced = false,
                                                    visitPercent = rawAnsweredCount // Пишем чистую сумму ответов вместо процентов!
                                                )
                                                dao.saveVisit(closedVisit)
                                            }
                                            activeVisitStoreId = null
                                            activeVisitId = null
                                            savedAnswers = emptyList()
                                            
                                            val unsyncedVisits = dao.getUnsyncedVisits()
                                            unsyncedCount = unsyncedVisits.size
                                            completedStoreIds = unsyncedVisits.filter { it.endedAt != null }.map { it.storeId1c }
                                            currentScreen = "MAIN"
                                        }
                                    }



                                )
                            } else {
                                currentScreen = "MAIN"
                            }
                        }

                        "MAIN" -> {

                                                       
                            LaunchedEffect(visitSubscreenSource) {
                                try {
                                    val locManager = applicationContext.getSystemService(android.content.Context.LOCATION_SERVICE) as android.location.LocationManager
                                    val providers = locManager.getProviders(true)
                                    var bestLocation: android.location.Location? = null
                                    for (provider in providers) {
                                        val l = locManager.getLastKnownLocation(provider) ?: continue
                                        val currentBest = bestLocation 
                                        if (currentBest == null || l.accuracy < currentBest.accuracy) {
                                            bestLocation = l
                                        }
                                    }
                                    val finalLoc = bestLocation
                                    if (finalLoc != null) {
                                        deviceLatitude = finalLoc.latitude
                                        deviceLongitude = finalLoc.longitude
                                    }
                                } catch (e: SecurityException) {}
                            }



                            MainScreen(
                                manufacturersList = manufacturersList,
                                storeManufacturerLinks = storeManufacturerLinks,
                                meName = meName,
                                storesList = storesList,
                                activeVisitStoreId = activeVisitStoreId,
                                unsyncedCount = unsyncedCount,
                                completedStoreIds = completedStoreIds,
                                
                                initialTabInternal = visitSubscreenSource,
                                deviceLatitude = deviceLatitude,  
                                deviceLongitude = deviceLongitude,
                                                               
                                onStartVisitClick = { clickedStoreId, sourceTab ->
                                    visitSubscreenSource = sourceTab
                                    
                                    activeVisitStoreId = clickedStoreId
                                    currentScreen = "VISIT"

                                    lifecycleScope.launch {
                                        val initialActive = dao.getActiveVisit()
                                        if (initialActive != null) {
                                            val timeFormatter = java.text.SimpleDateFormat("yyyy-MM-dd HH:mm", java.util.Locale.US)
                                            try {
                                                // Очищаем серверную строку до чистых 16 символов
                                                val safeStartedAt = initialActive.startedAt
                                                    .replace("T", " ")
                                                    .let { if (it.length > 16) it.substring(0, 16) else it }

                                                val startedDate = timeFormatter.parse(safeStartedAt)
                                                val currentDate = java.util.Calendar.getInstance().time
                                                if (startedDate != null) {
                                                    val diffMs = currentDate.time - startedDate.time
                                                    val diffHours = diffMs / (1000 * 60 * 60)
                                                    
                                                    if (diffHours >= 24) {
                                                        val cal = java.util.Calendar.getInstance().apply { time = startedDate }
                                                        cal.add(java.util.Calendar.HOUR_OF_DAY, 24)
                                                        val targetEndedAt = timeFormatter.format(cal.time)
                                                        
                                                        dao.saveVisit(initialActive.copy(endedAt = targetEndedAt, isSynced = false))
                                                        
                                                        android.widget.Toast.makeText(
                                                            applicationContext,
                                                            "ℹ️ Предыдущий открытый визит автоматически завершен (истек лимит 24 часа).",
                                                            android.widget.Toast.LENGTH_LONG
                                                        ).show()
                                                    }
                                                }
                                            } catch (e: Exception) {}
                                        }

                                        val finalActiveVisit = dao.getActiveVisit()

                                        if (finalActiveVisit != null && finalActiveVisit.storeId1c == clickedStoreId) {
                                            activeVisitId = finalActiveVisit.id
                                            savedAnswers = dao.getAnswersForVisit(finalActiveVisit.id)
                                        } else {
                                            val unsyncedVisit = dao.getLastUnsyncedVisitForStore(clickedStoreId)
                                            if (unsyncedVisit != null) {
                                                activeVisitId = unsyncedVisit.id
                                                savedAnswers = dao.getAnswersForVisit(unsyncedVisit.id)
                                            } else {
                                                val timeFormatter = java.text.SimpleDateFormat("yyyy-MM-dd HH:mm", java.util.Locale.US)
                                                val liveStartedAt = timeFormatter.format(java.util.Calendar.getInstance().time)
                                                val newVisitId = java.util.UUID.randomUUID().toString()

                                                val connManager = applicationContext.getSystemService(android.content.Context.CONNECTIVITY_SERVICE) as android.net.ConnectivityManager
                                                val networkCapabilities = connManager.getNetworkCapabilities(connManager.activeNetwork)
                                                val liveIsOnline = networkCapabilities != null && networkCapabilities.hasCapability(android.net.NetworkCapabilities.NET_CAPABILITY_INTERNET)
                                                
                                                val calendar = java.util.Calendar.getInstance()
                                                val androidDay = calendar.get(java.util.Calendar.DAY_OF_WEEK)
                                                val currentDayOfWeek = if (androidDay == java.util.Calendar.SUNDAY) 7 else androidDay - 1

                                                val activeStoreBrands = dao.getManufacturersForStore(clickedStoreId)
                                                val liveIsScheduled = activeStoreBrands.any { manufacturer ->
                                                    manufacturer.workDaysStr.split(",")
                                                        .map { it.trim() }
                                                        .contains(currentDayOfWeek.toString())
                                                }

                                                  
                                                val newVisit = VisitEntity(
                                                    id = newVisitId, 
                                                    storeId1c = clickedStoreId,
                                                    startedAt = liveStartedAt, 
                                                    endedAt = null,
                                                    isScheduled = liveIsScheduled, 
                                                    isOnline = liveIsOnline,
                                                    latitude = deviceLatitude ?: 0.0,   
                                                    longitude = deviceLongitude ?: 0.0, 
                                                    isSynced = false
                                                )
                                                dao.saveVisit(newVisit)
                                                activeVisitId = newVisitId
                                                savedAnswers = emptyList()
                                            }
                                        }
                                    }
                                },

                               
                                onUploadReportsClick = { onResult ->
                                    lifecycleScope.launch {
                                       
                                        
                                        val rawCredentials = "$mePhone:$mePassword"
                                        val base64Header = "Basic " + android.util.Base64.encodeToString(
                                            rawCredentials.toByteArray(), android.util.Base64.NO_WRAP
                                        )
                                        
                                        val syncedCount = dataRepository.flushOfflineQueue(base64Header, cacheDir)
                                        
                                        val unsyncedVisits = dao.getUnsyncedVisits()
                                        unsyncedCount = unsyncedVisits.size
                                        
                                        val todayPrefix = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.US).format(java.util.Calendar.getInstance().time)
                                        completedStoreIds = dao.getVisitedStoreIdsForToday(todayPrefix)
                                        
                                        testJsonDiagnosticData = null
                                        onResult(syncedCount)
                                    }
                                },

                                
                                onSyncDatabaseClick = { onResult ->
                                    lifecycleScope.launch {
                                        val rawCredentials = "$mePhone:$mePassword"
                                        val base64Header = "Basic " + android.util.Base64.encodeToString(
                                            rawCredentials.toByteArray(), android.util.Base64.NO_WRAP
                                        )
                                        val syncResult = dataRepository.performMorningSync(base64Header)
                                        
                                        storesList = dao.getAllStores()
                                        questionsList = dao.getAllQuestions()
                                        manufacturersList = dao.getAllManufacturers()
                                        storeManufacturerLinks = dao.getAllStoreManufacturerLinks()
                                        
                                        onResult(syncResult.isSuccess) 
                                    }
                                    
                                },
                                onLogout = {
                                    isAuthenticated = false
                                    meName = ""
                                    currentScreen = "LOGIN"
                                }
                            )
                        }

                    } // Конец конструкции when (currentScreen)

                                       
                    val diagnosticText = testJsonDiagnosticData
                    if (diagnosticText != null) {
                        androidx.compose.ui.window.Dialog(
                            onDismissRequest = { testJsonDiagnosticData = null }
                        ) {
                            androidx.compose.material3.Card(
                                modifier = Modifier
                                    .fillMaxSize()
                                    .padding(16.dp),
                                shape = androidx.compose.foundation.shape.RoundedCornerShape(12.dp)
                            ) {
                                androidx.compose.foundation.layout.Column(
                                    modifier = Modifier
                                        .fillMaxSize()
                                        .padding(16.dp)
                                ) {
                                    androidx.compose.material3.Text(
                                        text = "Инспектор SFA",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = androidx.compose.ui.text.font.FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.padding(bottom = 8.dp)
                                    )
                                    
                                    androidx.compose.foundation.layout.Box(
                                        modifier = Modifier
                                            .weight(1f)
                                            .fillMaxWidth()
                                            .background(androidx.compose.ui.graphics.Color(0xFFF5F5F5))
                                            .padding(8.dp)
                                    ) {
                                        androidx.compose.foundation.lazy.LazyColumn(
                                            modifier = Modifier.fillMaxSize()
                                        ) {
                                            item {
                                                androidx.compose.material3.Text(
                                                    text = diagnosticText,
                                                    style = MaterialTheme.typography.bodySmall,
                                                    fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                                                    color = androidx.compose.ui.graphics.Color(0xFF333333)
                                                )
                                            }
                                        }
                                    }
                                    
                                    androidx.compose.material3.Button(
                                        onClick = { testJsonDiagnosticData = null },
                                        modifier = Modifier
                                            .align(androidx.compose.ui.Alignment.CenterHorizontally)
                                            .padding(top = 12.dp)
                                    ) {
                                        androidx.compose.material3.Text("Закрыть инспектор")
                                    }
                                }
                            }
                        }
                    }

                } // Закрытие Surface
            } // Закрытие MaterialTheme
        } // Закрытие setContent
    } // Закрытие onCreate
           
   
  

} // Закрытие класса MainActivity

                                        