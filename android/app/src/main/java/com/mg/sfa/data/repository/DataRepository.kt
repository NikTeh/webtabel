package com.mg.sfa.data.repository

import com.mg.sfa.data.local.AppDao
import com.mg.sfa.data.local.StoreEntity
import com.mg.sfa.data.local.SkuEntity
import com.mg.sfa.data.local.MatrixEntity
import com.mg.sfa.data.local.QuestionEntity
import com.mg.sfa.data.local.ManufacturerEntity
import com.mg.sfa.data.remote.ApiService
import com.mg.sfa.data.remote.StartVisitRequestJson
import com.mg.sfa.data.remote.SyncReportsRequestJson
import com.mg.sfa.data.remote.ManufacturerReportJson
import com.mg.sfa.data.remote.AnswerDataJson
import com.mg.sfa.data.remote.FinishVisitRequestJson
import okhttp3.MediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody
import java.io.File
import okhttp3.MediaType.Companion.toMediaType


class DataRepository(
    private val dao: AppDao,
    private val api: ApiService
) {
    
    suspend fun performMorningSync(authHeader: String): Result<Unit> {
        return try {
            val response = api.syncData(authHeader)
            dao.deleteAllStores()

                        
            dao.deleteAllManufacturers()
            dao.insertManufacturers(response.manufacturers.map { 
                ManufacturerEntity(
                    it.id,
                    it.name,
                    it.work_days.joinToString(","),
                    it.duration_minutes
                )
            })

                        
            dao.deleteAllStoreManufacturerLinks()
            val remoteLinks = response.store_manufacturer_links.map {
                com.mg.sfa.data.local.StoreManufacturerLinkEntity(
                    storeId1c = it.store_id_1c,
                    manufacturerId = it.manufacturer_id
                )
            }
            dao.insertStoreManufacturerLinks(remoteLinks)

            
            dao.deleteAllHistoricalAnswers()
            response.historical_answers?.let { rawHistory ->
                val entities = rawHistory.map {
                    com.mg.sfa.data.local.HistoricalAnswerEntity(
                        storeId1c = it.store_id_1c,                       
                        questionId = it.question_id,
                        skuId = it.sku_id ?: "general",
                        valueText = it.value_text ?: "",
                        valueNumber = it.value_number
                    )
                }
                dao.insertHistoricalAnswers(entities)
            }




            
            dao.insertStores(response.stores.map { 
                StoreEntity(it.id_1c, it.region, it.city, it.retail_network, it.store_sign, it.store_format, it.address, it.sv_name, it.sv_phone, it.latitude ?: 0.0, it.longitude ?: 0.0)
            })
            dao.insertSkus(response.skus.map { 
                SkuEntity(it.id, it.manufacturer_id, it.name, it.brand, it.category)
            })
            dao.insertMatrices(response.matrices.map { 
                 MatrixEntity(it.sku_id, it.retail_network, it.store_format, it.is_available)
            })
            
             val activeManufacturerIds = response.questions.map { it.manufacturer_id }.distinct()
            
            for (mId in activeManufacturerIds) {
                dao.deleteQuestionsByManufacturer(mId)
            }

            // 3. И только на чистый диск записываем актуальную максимальную версию анкеты
            dao.insertQuestions(response.questions.map { 
                QuestionEntity(
                    id = it.id, 
                    manufacturerId = it.manufacturer_id,
                    questionnaireId = it.questionnaire_id, 
                    section = it.section, 
                    parentQuestionId = it.parent_question_id, 
                    questionText = it.question_text, 
                    responseType = it.response_type, 
                    availableOptions = it.available_options.joinToString("|"), 
                    sortOrder = it.sort_order,
                    conditionBranch = it.condition_branch,
                    conditionOperator = it.condition_operator,
                    conditionValueText = it.condition_value_text?.joinToString("|") ?: "",
                    conditionValueNumeric = it.condition_value_numeric,
                    allowGallery = it.allow_gallery,
                    dependsOnQuestionId = it.depends_on_question_id,
                    dependsOnOperator = it.depends_on_operator,
                    dependsOnValue = it.depends_on_value
                )
            })


            //



            //
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    // 2. Боевая сквозная пакетная отправка оффлайн-очереди на Go-сервер
    suspend fun flushOfflineQueue(authHeader: String, cacheDir: File): Int {
        val unsyncedVisits = dao.getUnsyncedVisits()
        var uploadCount = 0

        for (visit in unsyncedVisits) {
            try {
                // 1. Физическая отправка старта визита (/api/v1/me/visits/start)
                val startReq = StartVisitRequestJson(
                    id = visit.id,
                    store_id_1c = visit.storeId1c,
                    started_at = visit.startedAt.replace(" ", "T") + ":00+03:00",
                    is_scheduled = visit.isScheduled,
                    is_online = visit.isOnline,
                    start_latitude = if (visit.latitude != 0.0) visit.latitude else null,
                    start_longitude = if (visit.longitude != 0.0) visit.longitude else null
                )
                api.startVisit(authHeader, startReq)

                val answers = dao.getAnswersForVisit(visit.id)
                if (answers.isNotEmpty()) {
                    val questionIds: List<String> = answers.map { it.questionId }
                    val questions = dao.getQuestionsByIds(questionIds)
                    val questionMap = questions.associateBy { it.id }

                    // 2. Группировка ответов по честным парам (ID Производителя + Живой ID Анкеты 3 или 9 из Room)
                    val reportsPayload = answers.groupBy { ans ->
                        val q = questionMap[ans.questionId.toIntOrNull()]
                        Pair(q?.manufacturerId ?: 0, q?.questionnaireId ?: 0)
                    }.filter { it.key.first != 0 && it.key.second != 0 }
                    
                    reportsPayload.forEach { (key, answerList) ->
                        val manufacturerReport = ManufacturerReportJson(
                            manufacturer_id = key.first,
                            questionnaire_id = key.second, // Передаем реальный ID анкеты в Postgres
                            answers = answerList.map { ans ->
                                val q = questionMap[ans.questionId.toIntOrNull()]
                                val isNumberType = q?.responseType == "number"

                                AnswerDataJson(
                                    question_id = ans.questionId.toIntOrNull() ?: 0,
                                    sku_id = ans.skuId.ifEmpty { null },
                                    // Жесткое разделение типов полей для строгого валидатора бэкенда Go
                                    value_text = if (isNumberType || ans.valueText.isEmpty()) null else ans.valueText,
                                    value_number = if (isNumberType) ans.valueNumber else null
                                )
                            }
                        )

                        val syncReportsReq = SyncReportsRequestJson(
                            visit_id = visit.id,
                            reports = listOf(manufacturerReport)
                        )
                        // Пакетная отправка ответов (/api/v1/me/visits/reports/sync)
                        api.syncVisitReports(authHeader, syncReportsReq)
                    }

 
               
                answers.forEach { ans ->
                    if (ans.valueText.contains("visits/") && !ans.isPhotoUploaded) {
                        
                        val photoKeysList = ans.valueText.split("|").filter { it.isNotEmpty() }
                        var allPhotosInAnswerUploaded = true

                        for (photoKey in photoKeysList) {
                            
                            val finalFileToUpload = File(File(cacheDir.parentFile, "files"), photoKey)
                            
                            if (finalFileToUpload.exists()) {
                                try {                                                                     
                                    val textType = "text/plain".toMediaType()
                                    val imageType = "image/jpeg".toMediaType()

                                    
                                    val answerIdBody = RequestBody.create(textType, ans.questionId)
                                    val fileRequestBody = RequestBody.create(imageType, finalFileToUpload)
                                    
                                    // ДОБАВЛЕНО ДЛЯ ПРОДАKШHА: Передаем полный виртуальный оффлайн-ключ (visits/22/имя.jpg)
                                    val photoKeyBody = RequestBody.create(textType, photoKey)
                                    
                                  
                                    val photoPart = MultipartBody.Part.createFormData("photo", finalFileToUpload.name, fileRequestBody)
                                    
                                    // Обновляем вызов интерфейса (добавляем photoKeyBody в параметры)
                                    api.uploadPhoto(authHeader, answerIdBody, photoKeyBody, photoPart)
                                    
                                  
                                    finalFileToUpload.delete()
                                } catch (e: Exception) {
                                    allPhotosInAnswerUploaded = false
                                    break
                                }
                            }
                        }

                        if (allPhotosInAnswerUploaded && photoKeysList.isNotEmpty()) {
                            dao.saveAnswer(ans.copy(isPhotoUploaded = true))
                        }
                    }
                }


                }

                // 4. Физическая отправка закрытия визита по времени (/api/v1/me/visits/finish)
                val finishReq = FinishVisitRequestJson(
                    id = visit.id,
                    finished_at = (visit.endedAt ?: visit.startedAt).replace(" ", "T") + ":00+03:00"
                )
                api.finishVisit(authHeader, finishReq)

                // Если вся цепочка HTTP-запросов прошла без сбоев — помечаем визит в SQLite как синхронизированный
                dao.saveVisit(visit.copy(isSynced = true))
                uploadCount++

            } catch (e: Exception) {
                // При любом сетевом сбое (HTTP 400, 500) прерываем обработку этого визита и идем к следующему
                continue
            }
        }
        return uploadCount
    }
}
