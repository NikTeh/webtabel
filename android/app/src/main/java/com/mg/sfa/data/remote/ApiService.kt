package com.mg.sfa.data.remote

import okhttp3.MultipartBody
import okhttp3.RequestBody
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.POST
import retrofit2.http.Multipart
import retrofit2.http.Part

data class LoginResponse(val status: String, val role: String, val name: String)

data class RemoteHistoricalAnswer(
    val store_id_1c: String, // <--- ДОБАВЛЕHО
    val question_id: Int,
    val sku_id: String?,
    val value_text: String?,
    val value_number: Double?
)

data class SyncDataResponse(
    val stores: List<RemoteStore>,
    val skus: List<RemoteSku>,
    val matrices: List<RemoteMatrix>,
    val questions: List<RemoteQuestion>,
    val manufacturers: List<RemoteManufacturer>,             
    val store_manufacturer_links: List<RemoteStoreManufacturerLink>,// <--- Получение кэша из Go
    val historical_answers: List<RemoteHistoricalAnswer>? 
)

data class RemoteStore(val id_1c: String, val region: String, val city: String, val retail_network: String, val store_sign: String, val store_format: String, val address: String, val sv_name: String, val sv_phone: String, val latitude: Double?, val longitude: Double?)
data class RemoteSku(val id: String, val manufacturer_id: Int, val name: String, val brand: String, val category: String)
data class RemoteMatrix(val sku_id: String, val retail_network: String, val store_format: String, val is_available: Boolean)
data class RemoteQuestion(
    val id: Int,                              // УНИФИКАЦИЯ ТИПОВ: Числовой стандарт
    val manufacturer_id: Int, 
    val section: String, 
    val parent_question_id: Int?,             // УНИФИКАЦИЯ ТИПОВ: Числовой стандарт
    val question_text: String, 
    val response_type: String, 
    val available_options: List<String>, 
    val sort_order: Int, 
    val questionnaire_id: Int,
    val condition_branch: Boolean?,
    val condition_operator: String?,
    val condition_value_text: List<String>?,
    val condition_value_numeric: Double?,
    val allow_gallery: Boolean,
    val depends_on_question_id: Int?,         // ИСПРАВЛЕНО: Безопасный nullable тип
    val depends_on_operator: String?,         // ИСПРАВЛЕНО: Безопасный nullable тип // ИСПРАВЛЕНО: Безопасный nullable тип
    val depends_on_value: String?             
)


data class RemoteManufacturer(
    val id: Int,
    val name: String,
    val work_days: List<Int>,         // Массив дней недели {1,3,5} напрямую из PostgreSQL // Живой тайминг обслуживания
    val duration_minutes: Int         
)

data class RemoteStoreManufacturerLink(
    val store_id_1c: String,
    val manufacturer_id: Int
)


data class StartVisitRequestJson(val id: String, val store_id_1c: String, val started_at: String, val is_scheduled: Boolean, val is_online: Boolean, val start_latitude: Double?, val start_longitude: Double?)

data class AnswerDataJson(val question_id: Int, val sku_id: String?, val value_text: String?, val value_number: Double?)
data class ManufacturerReportJson(val manufacturer_id: Int, val questionnaire_id: Int, val answers: List<AnswerDataJson>)
data class SyncReportsRequestJson(val visit_id: String, val reports: List<ManufacturerReportJson>)


data class FinishVisitRequestJson(val id: String, val finished_at: String)
data class GenericMobileResponse(val status: String, val message: String)

interface ApiService {
    @POST("/api/v1/auth/login")
    suspend fun login(@Header("Authorization") authHeader: String): LoginResponse

    @GET("/api/v1/me/sync")
    suspend fun syncData(@Header("Authorization") authHeader: String): SyncDataResponse

    @POST("/api/v1/me/visits/start")
    suspend fun startVisit(@Header("Authorization") authHeader: String, @Body request: StartVisitRequestJson): GenericMobileResponse

   
    @Multipart
    @POST("/api/v1/me/photos/upload")
    suspend fun uploadPhoto(
        @Header("Authorization") authHeader: String, 
        @Part("answer_id") answerId: RequestBody, 
        @Part("photo_key") photoKey: RequestBody, 
        @Part photo: MultipartBody.Part
    ): GenericMobileResponse

    
    @POST("/api/v1/me/visits/reports/sync")
    suspend fun syncVisitReports(@Header("Authorization") authHeader: String, @Body request: SyncReportsRequestJson): GenericMobileResponse

    @POST("/api/v1/me/visits/finish")
    suspend fun finishVisit(@Header("Authorization") authHeader: String, @Body request: FinishVisitRequestJson): GenericMobileResponse
}
