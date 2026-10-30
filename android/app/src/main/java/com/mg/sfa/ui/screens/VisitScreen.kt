package com.mg.sfa.ui.screens

import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import coil.compose.AsyncImage 

import android.app.DatePickerDialog
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.mg.sfa.data.local.QuestionEntity
import com.mg.sfa.data.local.AnswerEntity
import com.mg.sfa.data.local.SkuEntity
import com.mg.sfa.data.local.ManufacturerEntity
import com.mg.sfa.data.local.StoreManufacturerLinkEntity
import java.io.File
import java.util.Calendar
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.foundation.gestures.detectTapGestures


data class ManufacturerProgress(
    val id: Int,
    val name: String,
    val isFullyCompleted: Boolean,
    val answeredCount: Int,
    val totalCount: Int,
    val skuCount: Int,
    val totalQuestionsLoad: Int,
    val durationMinutes: Int
)

@OptIn(ExperimentalMaterial3Api::class, androidx.compose.foundation.ExperimentalFoundationApi::class)
@Composable
fun VisitScreen(
    storeId1c: String,
    storeSign: String,
    storeAddress: String,
    questionsList: List<QuestionEntity>,
    skusList: List<SkuEntity>,
    manufacturersList: List<ManufacturerEntity>,
    storeManufacturerLinks: List<StoreManufacturerLinkEntity>,
    savedAnswers: List<AnswerEntity>,
    onAnswerChange: (questionId: String, skuId: String, text: String, num: Double?) -> Unit,
    onBackToMainMenuClick: () -> Unit,
    onCloseVisitClick: () -> Unit
) {
    var selectedManufacturerId by remember { mutableStateOf<Int?>(null) }
    var selectedManufacturerName by remember { mutableStateOf("") }

    var activeCameraQuestionId by remember { mutableStateOf<String?>(null) }
    var activeCameraSkuId by remember { mutableStateOf("") }
    var activeCameraCurrentAnswer by remember { mutableStateOf("") }      

   
    val focusManager = androidx.compose.ui.platform.LocalFocusManager.current    

    
    val isQuestionVisible: (QuestionEntity, String) -> Boolean = remember(savedAnswers, questionsList) {
        { question, skuId ->
            var current: QuestionEntity? = question
            var visible = true
            
            
            if (question.dependsOnQuestionId != null) {
                val triggerId = question.dependsOnQuestionId.toString()
                val triggerAnswer = savedAnswers.find { it.questionId == triggerId && it.skuId == skuId }
                
                if (triggerAnswer == null || triggerAnswer.valueText.trim().isEmpty()) {
                    visible = false // Родительский вопрос не заполнен — скрываем зависимый
                } else {
                    val enteredNum = triggerAnswer.valueNumber ?: triggerAnswer.valueText.toDoubleOrNull()
                    val targetNum = question.dependsOnValue?.toDoubleOrNull()
                    val op = question.dependsOnOperator ?: "="
                    
                    if (enteredNum != null && targetNum != null) {
                        // Числовое математическое сравнение диапазонов (Остатки, Цены, Промо)
                        visible = when (op) {
                            "<=" -> enteredNum <= targetNum
                            ">=" -> enteredNum >= targetNum
                            "<"  -> enteredNum < targetNum
                            ">"  -> enteredNum > targetNum
                            "!=" -> enteredNum != targetNum
                            else -> enteredNum == targetNum
                        }
                    } else {
                        // Обычное строгое текстовое сравнение строк (Списки выбора, Да/Нет)
                        val checkText = question.dependsOnValue?.trim() ?: ""
                        visible = if (op == "!=") {
                            !triggerAnswer.valueText.trim().equals(checkText, ignoreCase = true)
                        } else {
                            triggerAnswer.valueText.trim().equals(checkText, ignoreCase = true)
                        }
                    }
                }
            }
            
            // 2. СТАНДАРТНАЯ ПРОВЕРКА ВНУТРЕННИХ ИЕРАРХИЧЕСКИХ ВЕТОК ДЕРЕВА УСЛОВИЙ
            while (current?.parentQuestionId != null && visible) {
                val parentId = current.parentQuestionId
                val parentQuestion = questionsList.find { it.id == parentId }
                
                // Ищем ответ на родительский вопрос в текущем контексте SKU/Общий
                val parentAnswer = savedAnswers.find { it.questionId == parentId.toString() && it.skuId == skuId }
                
                if (parentAnswer == null || parentAnswer.valueText.trim().isEmpty()) {
                    visible = false
                } else if (parentQuestion != null) {
                    // Если родитель числового типа — выполняем математическое сравнение
                    if (parentQuestion.responseType == "number") {
                        val enteredNum = parentAnswer.valueNumber ?: parentAnswer.valueText.toDoubleOrNull()
                        val targetNum = current.conditionValueNumeric
                        val op = current.conditionOperator ?: "="
                        
                        if (enteredNum == null || targetNum == null) {
                            visible = false
                        } else {
                            visible = when (op) {
                                "<=" -> enteredNum <= targetNum
                                ">=" -> enteredNum >= targetNum
                                "<" -> enteredNum < targetNum
                                ">" -> enteredNum > targetNum
                                else -> enteredNum == targetNum
                            }
                        }
                    } else {
                        // Если родитель текстовый/список/дата — проверяем соответствие строк
                        val targetOptions = current.conditionValueText.split("|").map { it.trim() }.filter { it.isNotEmpty() }
                        if (targetOptions.isNotEmpty()) {
                            visible = targetOptions.contains(parentAnswer.valueText.trim())
                        } else if (current.conditionBranch != null) {
                            val isYes = parentAnswer.valueText.trim().equals("Да", ignoreCase = true)
                            visible = (isYes == current.conditionBranch)
                        }
                    }
                }
                current = parentQuestion
            }
            visible
        }
    }

    // Перехватчик изменения ответов с каскадным сбросом дочерних веток
    val onAnswerChangeWithCascade: (String, String, String, Double?) -> Unit = { qId, sku, text, num ->
        onAnswerChange(qId, sku, text, num)
        
        val numericQId = qId.toIntOrNull()
        if (numericQId != null) {
            val children = questionsList.filter { it.parentQuestionId == numericQId || it.dependsOnQuestionId == numericQId }
            if (children.isNotEmpty()) {
                fun clearChildren(parentId: Int) {
                    questionsList.filter { it.parentQuestionId == parentId || it.dependsOnQuestionId == parentId }.forEach { child ->
                        onAnswerChange(child.id.toString(), sku, "", null)
                        clearChildren(child.id)
                    }
                }
                clearChildren(numericQId)
            }
        }
    }
   
        
    val manufacturerProgressList = remember(questionsList, savedAnswers, skusList, manufacturersList, storeManufacturerLinks, storeId1c) {
        val storeActiveLinks = storeManufacturerLinks.filter { it.storeId1c == storeId1c }
        if (storeActiveLinks.isEmpty() || questionsList.isEmpty()) {
            emptyList()
        } else {
            storeActiveLinks.map { link ->
                val mId = link.manufacturerId
                val dbMan = manufacturersList.find { it.id == mId }
                val realName = dbMan?.name ?: "Производитель (ID: $mId)"
                val durationMinutes = dbMan?.durationMinutes ?: 30

                val manufacturerSkus = skusList.filter { it.manufacturerId == mId }
                val mQuestions = questionsList.filter { it.manufacturerId == mId }
                
                var totalQuestionsLoad = 0
                var answeredCount = 0

                // Считаем общие вопросы точки
                val generalQuestions = mQuestions.filter { it.section == "general" }
                generalQuestions.forEach { question ->
                    if (isQuestionVisible(question, "")) {
                        totalQuestionsLoad++
                        val hasAnswer = savedAnswers.any { it.questionId == question.id.toString() && it.skuId.isEmpty() && it.valueText.trim().isNotEmpty() }
                        if (hasAnswer) answeredCount++
                    }
                }

                // Считаем SKU-матрицу товаров
                val skuQuestions = mQuestions.filter { it.section == "sku" }
                manufacturerSkus.forEach { sku ->
                    skuQuestions.forEach { question ->
                        if (isQuestionVisible(question, sku.id)) {
                            totalQuestionsLoad++
                            val hasAnswer = savedAnswers.any { it.questionId == question.id.toString() && it.skuId == sku.id && it.valueText.trim().isNotEmpty() }
                            if (hasAnswer) answeredCount++
                        }
                    }
                }

                val isCompleted = totalQuestionsLoad > 0 && (answeredCount >= totalQuestionsLoad)

                ManufacturerProgress(
                    id = mId, name = realName, isFullyCompleted = isCompleted,
                    answeredCount = answeredCount, totalCount = totalQuestionsLoad,
                    skuCount = manufacturerSkus.size, totalQuestionsLoad = totalQuestionsLoad,
                    durationMinutes = durationMinutes
                )
            }
        }
    }

    val targetCamQuestionId = activeCameraQuestionId
    if (targetCamQuestionId != null) {
        val androidContext = LocalContext.current
        CameraCaptureScreen(
            onPhotoCaptured = { rawFile ->
                val targetRelativePath = "visits/${targetCamQuestionId}/${activeCameraSkuId.ifEmpty { "gen" }}_${System.currentTimeMillis()}.jpg"
                val compressedFile = com.mg.sfa.util.SecurityUtils.compressImageFile(
                    context = androidContext,
                    sourceFile = rawFile,
                    targetRelativePath = targetRelativePath
                )

                if (compressedFile != null && compressedFile.exists()) {
                    val finalPhotoString = if (activeCameraCurrentAnswer.isEmpty()) {
                        targetRelativePath
                    } else {
                        "$activeCameraCurrentAnswer|$targetRelativePath"
                    }
                    onAnswerChangeWithCascade(targetCamQuestionId, activeCameraSkuId, finalPhotoString, null)
                }
                activeCameraQuestionId = null
            },
            onCancel = { activeCameraQuestionId = null }
        )
        return
    }

    if (selectedManufacturerId == null) {
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            item {
                Text(text = storeSign, fontSize = 24.sp, fontWeight = FontWeight.ExtraBold, color = MaterialTheme.colorScheme.primary)
                Text(text = "Активный визит (Код: $storeId1c)", fontSize = 14.sp, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.outline)
                Text(text = storeAddress, fontSize = 13.sp, color = MaterialTheme.colorScheme.outline, modifier = Modifier.padding(top = 2.dp, bottom = 4.dp))
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Text(text = "Объем задач на торговой точке:", fontWeight = FontWeight.Bold, fontSize = 14.sp)
            }

            if (manufacturerProgressList.isEmpty()) {
                item {
                    Box(modifier = Modifier.fillMaxWidth().height(100.dp), contentAlignment = Alignment.Center) {
                        Text("Нет доступных контрактов. Проведите синхронизацию.", color = MaterialTheme.colorScheme.outline, fontSize = 13.sp)
                    }
                }
            } else {
                items(manufacturerProgressList) { client ->
                    val backgroundColor = if (client.isFullyCompleted) Color(0xFFE6F4EA) else Color(0xFFFDF2F2)
                    val contentColor = if (client.isFullyCompleted) Color(0xFF137333) else Color(0xFF981B1B)

                    Card(
                        modifier = Modifier.fillMaxWidth().clickable { 
                            selectedManufacturerId = client.id
                            selectedManufacturerName = client.name
                        },
                        colors = CardDefaults.cardColors(containerColor = backgroundColor)
                    ) {
                        Column(modifier = Modifier.padding(14.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(text = client.name, fontSize = 17.sp, fontWeight = FontWeight.Bold, color = contentColor)
                                Text(text = if (client.isFullyCompleted) "● Выполнено" else "○ В процессе", fontWeight = FontWeight.Bold, fontSize = 11.sp, color = contentColor)
                            }
                            Divider(modifier = Modifier.padding(vertical = 8.dp), color = contentColor.copy(alpha = 0.15f))
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                                Column {
                                    Text(text = "🕒 Тайминг: ${client.durationMinutes} мин", fontSize = 12.sp, fontWeight = FontWeight.Medium, color = Color(0xFF333333))
                                    Text(text = "📦 Матрица: ${client.skuCount} SKU", fontSize = 12.sp, fontWeight = FontWeight.Medium, color = Color(0xFF333333))
                                }
                                Column {
                                    Text(text = "📋 Базовых задач: ${client.totalCount}", fontSize = 12.sp, fontWeight = FontWeight.Medium, color = Color(0xFF333333))
                                    Text(text = "📊 Учтено ответов: ${client.answeredCount}", fontSize = 12.sp, fontWeight = FontWeight.Bold, color = contentColor)
                                }
                            }
                        }
                    }
                }
            }

            

            item {
                var showEmptyWarningDialog by remember { mutableStateOf(false) }

                if (showEmptyWarningDialog) {
                    AlertDialog(
                        onDismissRequest = { showEmptyWarningDialog = false },
                        title = { Text("⚠️ Пустой отчет по визиту") },
                        text = { Text("Вы не заполнили ни одного ответа в анкетах производителей. Визит будет сохранен в системе как пустой (Только выкладка товаров). Вы сможете дозаполнить анкеты позже через кнопку 'Редактировать отчет'. Завершить визит?") },
                        confirmButton = {
                            Button(
                                onClick = {
                                    showEmptyWarningDialog = false
                                    onCloseVisitClick()
                                },
                                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
                            ) { Text("Да, закрыть точку") }
                        },
                        dismissButton = {
                            OutlinedButton(onClick = { showEmptyWarningDialog = false }) { Text("Отмена") }
                        }
                    )
                }

                Spacer(modifier = Modifier.height(8.dp))
                OutlinedButton(onClick = onBackToMainMenuClick, modifier = Modifier.fillMaxWidth()) {
                    Text("Свернуть черновик и вернуться в Главное меню")
                }
                Spacer(modifier = Modifier.height(8.dp))
                Button(
                    onClick = { 
                        // Считаем количество реальных ответов текущего визита Николая
                        val currentVisitAnswersCount = savedAnswers.count { it.valueText.trim().isNotEmpty() }
                        if (currentVisitAnswersCount == 0) {
                            showEmptyWarningDialog = true // Вызываем окно предупреждения
                        } else {
                            onCloseVisitClick() // Обычное закрытие, если ответы есть
                        }
                    }, 
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error), 
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("Завершить визит в магазине", fontWeight = FontWeight.Bold)
                }
            }

        }
    } else {
        val currentManufacturerId = selectedManufacturerId!!
        val generalQuestions = remember(questionsList, currentManufacturerId) {
            questionsList.filter { it.manufacturerId == currentManufacturerId && it.section == "general" }
        }
        val skuQuestions = remember(questionsList, currentManufacturerId) {
            questionsList.filter { it.manufacturerId == currentManufacturerId && it.section == "sku" }
        }
        val currentManufacturerSkus = remember(skusList, currentManufacturerId) {
            skusList.filter { it.manufacturerId == currentManufacturerId }
        }
        val skusGroupedByCategory = remember(currentManufacturerSkus) {
            currentManufacturerSkus.groupBy { it.category }
        }

        
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp)
               
                .pointerInput(Unit) {
                    detectTapGestures(onTap = {
                        focusManager.clearFocus()                        
                    })
                },
            verticalArrangement = Arrangement.SpaceBetween
        ) {
            Column(modifier = Modifier.fillMaxWidth().weight(1f, fill = false)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(text = selectedManufacturerName, fontSize = 18.sp, fontWeight = FontWeight.Bold)
                        Text(text = "Контроль показателей", fontSize = 12.sp, color = Color.Gray)
                    }
                    Button(
                        onClick = { selectedManufacturerId = null }, 
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.secondary), 
                        contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp), 
                        modifier = Modifier.height(36.dp)
                    ) {
                        Text("← К клиентам", fontSize = 12.sp, fontWeight = FontWeight.Bold)
                    }
                }

                Divider(modifier = Modifier.padding(vertical = 12.dp))

                if (generalQuestions.isEmpty() && skuQuestions.isEmpty()) {
                    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer), modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp)) {
                        Text(text = "📋 По данному бренду требуется только выкладка товара на полку согласно планограмме торговой точки. Заполнение анкеты не требуется.", color = MaterialTheme.colorScheme.onPrimaryContainer, fontSize = 14.sp, fontWeight = FontWeight.Medium, modifier = Modifier.padding(16.dp))
                    }
                } else {
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        
                        // ... 3-4 строки исходного окружающего контекста для Ctrl+F:
                        if (generalQuestions.isNotEmpty()) {
                            val visibleGenerals = generalQuestions.filter { isQuestionVisible(it, "") }
                            if (visibleGenerals.isNotEmpty()) {
                                item { Text(text = "📋 Общие вопросы по точке", fontWeight = FontWeight.Bold, fontSize = 14.sp, color = MaterialTheme.colorScheme.primary) }
                                items(visibleGenerals) { question ->
                                    val currentAnswer = savedAnswers.find { it.questionId == question.id.toString() && it.skuId.isEmpty() }?.valueText ?: ""
                                    RenderQuestionRow(
                                        storeId1c = storeId1c, 
                                        question = question, 
                                        skuId = "", 
                                        currentAnswer = currentAnswer, 
                                        savedAnswers = savedAnswers, // <--- ДОБАВЛЕНО СЮДА
                                        onAnswerChange = onAnswerChangeWithCascade, 
                                        onLaunchCamera = { qID, sID, ansStr ->
                                            activeCameraQuestionId = qID
                                            activeCameraSkuId = sID
                                            activeCameraCurrentAnswer = ansStr
                                        }
                                    )
                                }
                            }
                        }
// ...


                        if (currentManufacturerSkus.isNotEmpty() && skuQuestions.isNotEmpty()) {
                            skusGroupedByCategory.forEach { (categoryName, skusInCat) ->
                                var hasVisibleQuestionsInCat = false
                                skusInCat.forEach { sku ->
                                    if (skuQuestions.any { isQuestionVisible(it, sku.id) }) {
                                        hasVisibleQuestionsInCat = true
                                    }
                                }

                                if (hasVisibleQuestionsInCat) {
                                    item {
                                        Spacer(modifier = Modifier.height(8.dp))
                                        Text(text = "📦 КАТЕГОРИЯ: ${categoryName.uppercase()}", fontWeight = FontWeight.ExtraBold, fontSize = 13.sp, color = Color(0xFF333333), modifier = Modifier.background(Color(0xFFEFEFEF)).padding(4.dp).fillMaxWidth())
                                    }

                                                                                skusInCat.forEach { sku ->
                                        val visibleSkuQuestions = skuQuestions.filter { isQuestionVisible(it, sku.id) }
                                        if (visibleSkuQuestions.isNotEmpty()) {
                                            item { Text(text = sku.name, fontWeight = FontWeight.Bold, fontSize = 14.sp, color = MaterialTheme.colorScheme.secondary, modifier = Modifier.padding(top = 4.dp)) }
                                            items(visibleSkuQuestions) { question ->
                                                val currentAnswer = savedAnswers.find { it.questionId == question.id.toString() && it.skuId == sku.id }?.valueText ?: ""
                                                Row(modifier = Modifier.padding(start = 12.dp)) {
                                                    RenderQuestionRow(
                                                        storeId1c = storeId1c, 
                                                        question = question, 
                                                        skuId = sku.id, 
                                                        currentAnswer = currentAnswer, 
                                                        savedAnswers = savedAnswers, 
                                                        onAnswerChange = onAnswerChangeWithCascade, 
                                                        onLaunchCamera = { qID, sID, ansStr ->
                                                            activeCameraQuestionId = qID
                                                            activeCameraSkuId = sID
                                                            activeCameraCurrentAnswer = ansStr
                                                        }
                                                    )
                                                }
                                            }
                                        }
                                    }
// ...

                                }
                            }
                        }
                    }
                }
            }

            Button(onClick = { selectedManufacturerId = null }, modifier = Modifier.fillMaxWidth().padding(top = 12.dp)) {
                Text("Сохранить черновик анкеты")
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RenderQuestionRow(
    storeId1c: String,
    question: QuestionEntity,
    skuId: String,
    currentAnswer: String,
    savedAnswers: List<AnswerEntity>, // Передаем живые ответы текущей смены
    onAnswerChange: (String, String, String, Double?) -> Unit,
    onLaunchCamera: (String, String, String) -> Unit
) {
    val context = LocalContext.current

    val galleryLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent()
    ) { uri: android.net.Uri? ->
        if (uri != null) {
            val finalPath = com.mg.sfa.util.GalleryImporter.importPhoto(context, uri, question.id, skuId)
            if (finalPath != null) {
                val finalPhotoString = if (currentAnswer.isEmpty()) {
                    finalPath
                } else {
                    "$currentAnswer|$finalPath"
                }
                onAnswerChange(question.id.toString(), skuId, finalPhotoString, null)
            } else {
                android.widget.Toast.makeText(context, "❌ Ошибка импорта фото из галереи", android.widget.Toast.LENGTH_SHORT).show()
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(if (currentAnswer.isNotEmpty()) Color(0xFFF1F8F5) else Color.Transparent, shape = MaterialTheme.shapes.small)
            .padding(4.dp)
    ) {
        Text(text = "${question.sortOrder}. ${question.questionText}", fontWeight = FontWeight.Medium, fontSize = 13.sp, modifier = Modifier.padding(bottom = 4.dp))

        val dao = com.mg.sfa.data.local.AppDatabase.getDatabase(context).appDao()
        var historicalPlaceholder by remember(storeId1c, question.id, skuId) { mutableStateOf("") }

        LaunchedEffect(storeId1c, question.id, skuId, savedAnswers) {
            val targetSkuKey = skuId.ifEmpty { "general" }
            
            // 1. Проверяем наличие ответа в RAM буфере за текущие сутки
            val ramAnswer = savedAnswers.find { it.questionId == question.id.toString() && it.skuId == skuId }
            if (ramAnswer != null && ramAnswer.valueText.trim().isNotEmpty()) {
                historicalPlaceholder = ramAnswer.valueText
            } else {
                // 2. Если пустой — подтягиваем утренний исторический кэш
                val pastData = dao.getHistoricalAnswer(storeId1c, question.id, targetSkuKey)
                pastData?.let {
                    historicalPlaceholder = if (question.responseType == "number" && it.valueNumber != null) {
                        it.valueNumber.toString()
                    } else {
                        it.valueText
                    }
                }
            }
        }

        when (question.responseType) {
            "text" -> {
                val defaultHint = historicalPlaceholder.ifEmpty { "Введите ответ..." }
                BufferedAnswerField(initialValue = currentAnswer, placeholderText = defaultHint, isHistoricalHint = historicalPlaceholder.isNotEmpty(), keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Text), singleLine = false, onFocusLost = { finalValue -> onAnswerChange(question.id.toString(), skuId, finalValue, null) })
            }
            "number" -> {
                val defaultHint = historicalPlaceholder.ifEmpty { "0" }
                BufferedAnswerField(initialValue = currentAnswer, placeholderText = defaultHint, isHistoricalHint = historicalPlaceholder.isNotEmpty(), keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number), singleLine = true, onFocusLost = { finalValue -> onAnswerChange(question.id.toString(), skuId, finalValue, finalValue.toDoubleOrNull()) })
            }
            "date" -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 4.dp)
                        .background(Color(0xFFF5F5F5), shape = MaterialTheme.shapes.small)
                        .clickable {
                            val calendar = Calendar.getInstance()
                            val dpd = DatePickerDialog(
                                context,
                                { _, year, month, dayOfMonth ->
                                    val formattedDate = String.format("%04d-%02d-%02d", year, month + 1, dayOfMonth)
                                    onAnswerChange(question.id.toString(), skuId, formattedDate, null)
                                },
                                calendar.get(Calendar.YEAR),
                                calendar.get(Calendar.MONTH),
                                calendar.get(Calendar.DAY_OF_MONTH)
                            )
                            dpd.show()
                        }
                        .padding(12.dp)
                ) {
                    Text(
                        text = if (currentAnswer.isNotEmpty()) "📅 $currentAnswer" else "📅 Выберите дату из календаря...",
                        fontSize = 13.sp,
                        color = if (currentAnswer.isNotEmpty()) Color.Black else Color.Gray
                    )
                }
            }
            "select", "condition" -> {
                val options = question.availableOptions.split("|").map { it.trim() }.filter { it.isNotEmpty() }
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    options.forEach { option ->
                        val isSelected = currentAnswer == option
                        FilterChip(selected = isSelected, onClick = { onAnswerChange(question.id.toString(), skuId, option, null) }, label = { Text(option, fontSize = 11.sp) })
                    }
                }
            }
            "multiselect" -> {
                val options = question.availableOptions.split("|").map { it.trim() }.filter { it.isNotEmpty() }
                val selectedOptions = currentAnswer.split("|").map { it.trim() }.filter { it.isNotEmpty() }.toMutableList()

                Column(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                    options.forEach { option ->
                        val isChecked = selectedOptions.contains(option)
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    if (isChecked) selectedOptions.remove(option) else selectedOptions.add(option)
                                    val finalStr = selectedOptions.joinToString("|")
                                    onAnswerChange(question.id.toString(), skuId, finalStr, null)
                                }
                                .padding(vertical = 4.dp)
                        ) {
                            Checkbox(
                                checked = isChecked,
                                onCheckedChange = null
                            )
                            Text(
                                text = option,
                                fontSize = 13.sp,
                                modifier = Modifier.padding(start = 8.dp)
                            )
                        }
                    }
                }
            }
            "photo" -> {
                Column(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        Button(
                            onClick = { onLaunchCamera(question.id.toString(), skuId, currentAnswer) },
                            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.tertiary)
                        ) {
                            Text("📸 Сделать фото", fontSize = 11.sp)
                        }

                        if (question.allowGallery) {
                            OutlinedButton(
                                onClick = { galleryLauncher.launch("image/*") },
                                contentPadding = PaddingValues(horizontal = 10.dp, vertical = 2.dp),
                                modifier = Modifier.height(32.dp)
                            ) {
                                Text("📁 Из галереи", fontSize = 11.sp, color = MaterialTheme.colorScheme.primary)
                            }
                        }
                    }

                    if (currentAnswer.isNotEmpty()) {
                        val photoPaths = currentAnswer.split("|").filter { it.isNotEmpty() }

                        Spacer(modifier = Modifier.height(8.dp))

                        LazyRow(
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            modifier = Modifier.fillMaxWidth().height(70.dp)
                        ) {
                            itemsIndexed(photoPaths) { idx, path ->
                                val imgFile = File(context.filesDir, path)
                                Box(modifier = Modifier.size(70.dp)) {
                                    AsyncImage(
                                        model = imgFile,
                                        contentDescription = "Превью фотоотчета",
                                        contentScale = ContentScale.Crop,
                                        modifier = Modifier.fillMaxSize().clip(RoundedCornerShape(6.dp)).background(Color.LightGray)
                                    )

                                    Box(
                                        modifier = Modifier
                                            .align(Alignment.TopEnd)
                                            .size(20.dp)
                                            .background(Color.Red.copy(alpha = 0.8f), shape = RoundedCornerShape(bottomStart = 6.dp))
                                            .clickable {
                                                try {
                                                    if (imgFile.exists()) {
                                                        imgFile.delete()
                                                    }
                                                    val updatedPaths = photoPaths.filterIndexed { i, _ -> i != idx }
                                                    val finalStr = updatedPaths.joinToString("|")
                                                    onAnswerChange(question.id.toString(), skuId, finalStr, null)
                                                } catch (e: Exception) {}
                                            },
                                        contentAlignment = Alignment.Center
                                    ) {
                                        Text("×", color = Color.White, fontSize = 14.sp, fontWeight = FontWeight.Bold, modifier = Modifier.padding(bottom = 2.dp))
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}


@Composable
fun BufferedAnswerField(
    initialValue: String,
    placeholderText: String,
    isHistoricalHint: Boolean = false,
    keyboardOptions: KeyboardOptions,
    singleLine: Boolean,
    onFocusLost: (String) -> Unit
) {
    var inputBuffer by remember(initialValue) { mutableStateOf(initialValue) }
    val focusManager = androidx.compose.ui.platform.LocalFocusManager.current

    OutlinedTextField(
        value = inputBuffer,
        onValueChange = { characters -> inputBuffer = characters },
        placeholder = { 
            Text(
                text = if (isHistoricalHint && inputBuffer.isEmpty()) "📋 Было: $placeholderText" else placeholderText, 
                fontSize = 12.sp,
                color = if (isHistoricalHint && inputBuffer.isEmpty()) Color.Gray else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f),
                fontWeight = if (isHistoricalHint) FontWeight.Bold else FontWeight.Normal
            ) 
        },
        // Добавляем промышленный trailingIcon быстрого копирования истории в один тап справа
        trailingIcon = {
            if (isHistoricalHint && inputBuffer.isEmpty()) {
                IconButton(
                    onClick = {
                        inputBuffer = placeholderText
                        onFocusLost(placeholderText) // Мгновенно фиксируем перенос в SQLite Room
                    },
                    modifier = Modifier.size(28.dp)
                ) {
                    Text("↩️", fontSize = 14.sp)
                }
            }
        },
        keyboardOptions = keyboardOptions.copy(
            imeAction = if (singleLine) androidx.compose.ui.text.input.ImeAction.Done else androidx.compose.ui.text.input.ImeAction.Default
        ),
        keyboardActions = androidx.compose.foundation.text.KeyboardActions(
            onDone = {
                onFocusLost(inputBuffer)
                focusManager.clearFocus()
            }
        ),
        singleLine = singleLine,
        maxLines = if (singleLine) 1 else 3,
        textStyle = LocalTextStyle.current.copy(fontSize = 13.sp),
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 2.dp)
            .onFocusChanged { focusState ->
                // Нажатие на поле теперь оставляет буфер чистым для ввода с нуля. 
                // Старая логика автозабивания при фокусе вырезана по ТЗ!
                if (!focusState.isFocused && inputBuffer != initialValue) {
                    onFocusLost(inputBuffer)
                }
            }
    )
}

