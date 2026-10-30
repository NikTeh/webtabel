package com.mg.sfa.data.local

import androidx.room.*

@Dao
interface AppDao {
    
    @Query("DELETE FROM local_visits WHERE id = :visitId")
    suspend fun deleteVisitById(visitId: String)

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertStores(stores: List<StoreEntity>)

    @Query("SELECT * FROM local_stores")
    suspend fun getAllStores(): List<StoreEntity>

    @Query("SELECT * FROM local_skus")
    suspend fun getAllSkus(): List<SkuEntity>
    
    
    @Query("DELETE FROM local_stores")
    suspend fun deleteAllStores()

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertSkus(skus: List<SkuEntity>)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMatrices(matrices: List<MatrixEntity>)

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertQuestions(questions: List<QuestionEntity>)

    @Query("SELECT * FROM local_questions WHERE manufacturerId = :manufacturerId ORDER BY sortOrder ASC")
    suspend fun getQuestionsForManufacturer(manufacturerId: Int): List<QuestionEntity>

    
    @Query("DELETE FROM local_questions WHERE manufacturerId = :manufacturerId")
    suspend fun deleteQuestionsByManufacturer(manufacturerId: Int)


    @Query("SELECT * FROM local_questions")
    suspend fun getAllQuestions(): List<QuestionEntity>

    @Query("SELECT * FROM local_questions WHERE id IN (:questionIds)")
    suspend fun getQuestionsByIds(questionIds: List<String>): List<QuestionEntity>

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun saveAnswer(answer: AnswerEntity)

    @Query("SELECT * FROM local_answers WHERE visitId = :visitId")
    suspend fun getAnswersForVisit(visitId: String): List<AnswerEntity>

    
    @Query("DELETE FROM local_answers WHERE visitId = :visitId AND questionId = :questionId AND skuId = :skuId")
    suspend fun deleteAnswer(visitId: String, questionId: String, skuId: String)

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun saveVisit(visit: VisitEntity)

    @Query("SELECT * FROM local_visits WHERE endedAt IS NULL LIMIT 1")
    suspend fun getActiveVisit(): VisitEntity?

    @Query("SELECT * FROM local_visits WHERE isSynced = 0 AND endedAt IS NOT NULL")
    suspend fun getUnsyncedVisits(): List<VisitEntity>

     @Query("SELECT * FROM local_visits WHERE storeId1c = :storeId AND endedAt IS NOT NULL AND isSynced = 0 ORDER BY startedAt DESC LIMIT 1")
    suspend fun getLastUnsyncedVisitForStore(storeId: String): VisitEntity?


    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun saveLocalUser(user: UserEntity)

    @Query("SELECT * FROM local_users WHERE phone = :phone LIMIT 1")
    suspend fun getLocalUser(phone: String): UserEntity?

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertManufacturers(manufacturers: List<ManufacturerEntity>)

    @Query("SELECT * FROM local_manufacturers ORDER BY id ASC")
    suspend fun getAllManufacturers(): List<ManufacturerEntity>

    @Query("DELETE FROM local_manufacturers")
    suspend fun deleteAllManufacturers()

    
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertStoreManufacturerLinks(links: List<StoreManufacturerLinkEntity>)

    @Query("SELECT * FROM local_store_manufacturer_links")
    suspend fun getAllStoreManufacturerLinks(): List<StoreManufacturerLinkEntity>

    @Query("DELETE FROM local_store_manufacturer_links")
    suspend fun deleteAllStoreManufacturerLinks()

        
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertHistoricalAnswers(answers: List<HistoricalAnswerEntity>)

    @Query("DELETE FROM local_historical_answers")
    suspend fun deleteAllHistoricalAnswers()

        
    @Query("SELECT storeId1c FROM local_visits WHERE endedAt IS NOT NULL AND startedAt LIKE :todayPrefix || '%'")
    suspend fun getVisitedStoreIdsForToday(todayPrefix: String): List<String>
    

    @Query("SELECT * FROM local_visits WHERE storeId1c = :storeId AND endedAt IS NOT NULL ORDER BY startedAt DESC LIMIT 1")
    suspend fun getLastClosedVisitForStore(storeId: String): VisitEntity?

        
    @Query("SELECT COUNT(*) FROM local_visits WHERE storeId1c = :storeId AND endedAt IS NOT NULL AND isSynced = 0")
    suspend fun getUnsyncedVisitsCountForStore(storeId: String): Int

    @Query("SELECT * FROM local_historical_answers WHERE storeId1c = :storeId AND questionId = :qId AND skuId = :sku LIMIT 1")
    suspend fun getHistoricalAnswer(storeId: String, qId: Int, sku: String): HistoricalAnswerEntity?

    
    @Query("""
        SELECT m.* FROM local_manufacturers m
        INNER JOIN local_store_manufacturer_links l ON m.id = l.manufacturerId
        WHERE l.storeId1c = :storeId
        ORDER BY m.name ASC
    """)
    suspend fun getManufacturersForStore(storeId: String): List<ManufacturerEntity>


}
