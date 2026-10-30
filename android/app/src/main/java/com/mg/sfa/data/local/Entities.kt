package com.mg.sfa.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "local_stores")
data class StoreEntity(
    @PrimaryKey val id1c: String,
    val region: String,
    val city: String,
    val retailNetwork: String,
    val storeSign: String,
    val storeFormat: String,
    val address: String,
    val supervisorName: String,
    val supervisorPhone: String,
    val latitude: Double = 0.0, 
    val longitude: Double = 0.0 
)

@Entity(tableName = "local_visits")
data class VisitEntity(
    @PrimaryKey val id: String,
    val storeId1c: String,
    val startedAt: String,
    val endedAt: String?,
    val isScheduled: Boolean,
    val isOnline: Boolean,
    val latitude: Double,
    val longitude: Double,
    val isSynced: Boolean = false,
    val visitPercent: Int = 0,  
    val isMockGps: Boolean = false,          
    val isGpsDisabled: Boolean = false,       
    val timeDeltaMs: Long = 0L,                
    val hasSuspiciousActivity: Boolean = false
) 

@Entity(tableName = "local_answers", primaryKeys = ["visitId", "questionId", "skuId"])
data class AnswerEntity(
    val visitId: String,
    val questionId: String,
    val skuId: String,
    val valueText: String,
    val valueNumber: Double?,
    val isPhotoUploaded: Boolean = false
)

@Entity(tableName = "local_questions")
data class QuestionEntity(
    @PrimaryKey val id: Int,                   
    val manufacturerId: Int,
    val questionnaireId: Int,                  
    val section: String,                       
    val parentQuestionId: Int?,               
    val questionText: String,
    val responseType: String,                  
    val availableOptions: String,              
    val sortOrder: Int,
    val conditionBranch: Boolean?,             
    val conditionOperator: String?,            
    val conditionValueText: String,            
    val conditionValueNumeric: Double?,        
    val allowGallery: Boolean,
    val dependsOnQuestionId: Int?,
    val dependsOnOperator: String?,           
    val dependsOnValue: String?                  
)


@Entity(tableName = "local_skus")
data class SkuEntity(
    @PrimaryKey val id: String,
    val manufacturerId: Int,
    val name: String,
    val brand: String,
    val category: String
)

@Entity(tableName = "local_matrices", primaryKeys = ["skuId", "retailNetwork", "storeFormat"])
data class MatrixEntity(
    val skuId: String,
    val retailNetwork: String,
    val storeFormat: String,
    val isAvailable: Boolean
)

@Entity(tableName = "local_users")
data class UserEntity(
    @PrimaryKey val phone: String,             
    val passwordHash: String,                  
    val fullName: String,                     
    val role: String                           
)

@Entity(tableName = "local_manufacturers")
data class ManufacturerEntity(
    @PrimaryKey val id: Int,                   
    val name: String,                          
    val workDaysStr: String,                   // Строка дней недели через запятую ("1,3,5") 
    val durationMinutes: Int                   
)

@Entity(
    tableName = "local_store_manufacturer_links",
    primaryKeys = ["storeId1c", "manufacturerId"] 
)
data class StoreManufacturerLinkEntity(
    val storeId1c: String,
    val manufacturerId: Int
)

@Entity(tableName = "local_historical_answers", primaryKeys = ["storeId1c", "questionId", "skuId"]) 
data class HistoricalAnswerEntity(
    val storeId1c: String, 
    val questionId: Int,
    val skuId: String, 
    val valueText: String,
    val valueNumber: Double?
)




