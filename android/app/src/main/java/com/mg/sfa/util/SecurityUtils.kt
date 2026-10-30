package com.mg.sfa.util

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.location.Location
import android.os.SystemClock
import java.io.File
import java.io.FileOutputStream

object SecurityUtils {

    fun compressImageFile(context: Context, sourceFile: File, targetRelativePath: String, targetQuality: Int = 75): File? {
        
        val bitmap = BitmapFactory.decodeFile(sourceFile.absolutePath) ?: return null
        
        val maxDimension = 1920
        val scaledBitmap = if (bitmap.width > maxDimension || bitmap.height > maxDimension) {
            val ratio = bitmap.width.toFloat() / bitmap.height.toFloat()
            val newWidth = if (ratio > 1) maxDimension else (maxDimension * ratio).toInt()
            val newHeight = if (ratio > 1) (maxDimension / ratio).toInt() else maxDimension
            Bitmap.createScaledBitmap(bitmap, newWidth, newHeight, true)
        } else {
            bitmap
        }

        val finalDestinationFile = File(context.filesDir, targetRelativePath)
        
        finalDestinationFile.parentFile?.mkdirs()

        try {
            FileOutputStream(finalDestinationFile).use { fos ->
                scaledBitmap.compress(Bitmap.CompressFormat.JPEG, targetQuality, fos)
            }
            
            if (bitmap != scaledBitmap) bitmap.recycle()
            scaledBitmap.recycle()
            
            if (sourceFile.exists()) {
                sourceFile.delete()
            }
            
            return finalDestinationFile
        } catch (e: Exception) {
            if (bitmap != scaledBitmap) bitmap.recycle()
            scaledBitmap.recycle()
            return null
        }
    }


    fun isLocationMocked(location: Location?): Boolean {
        if (location == null) return false
        return location.isFromMockProvider
    }

    fun getSecureTimestamp(): Long {
        return SystemClock.elapsedRealtime()
    }

     fun calculateDistanceMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
        val r = 6371000.0 // Средний радиус Земли в метрах
        val dLat = Math.toRadians(lat2 - lat1)
        val dLon = Math.toRadians(lon2 - lon1)
        val a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
                Math.cos(Math.toRadians(lat1)) * Math.cos(Math.toRadians(lat2)) *
                Math.sin(dLon / 2) * Math.sin(dLon / 2)
        val c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
        return r * c
    }

}


