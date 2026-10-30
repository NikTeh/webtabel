package com.mg.sfa.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase


@Database(
    entities = [
        StoreEntity::class, 
        SkuEntity::class, 
        MatrixEntity::class, 
        QuestionEntity::class, 
        VisitEntity::class, 
        AnswerEntity::class,
        UserEntity::class, 
        ManufacturerEntity::class,        
        StoreManufacturerLinkEntity::class,
        HistoricalAnswerEntity::class 
    ], 
    version = 1, 
    exportSchema = false
)
abstract class AppDatabase : RoomDatabase() {
    
    
    abstract fun appDao(): AppDao

    companion object {
        @Volatile
        private var INSTANCE: AppDatabase? = null

        
        fun getDatabase(context: Context): AppDatabase {
            return INSTANCE ?: synchronized(this) {
                val instance = Room.databaseBuilder(
                    context.applicationContext,
                    AppDatabase::class.java,
                    "mg_sfa_local.db" 
                )
                .fallbackToDestructiveMigration()
                .build()
                INSTANCE = instance
                instance
            }
        }
    }
}
