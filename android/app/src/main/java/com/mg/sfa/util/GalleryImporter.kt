package com.mg.sfa.util

import android.content.Context
import android.net.Uri
import java.io.File

object GalleryImporter {
    
    fun importPhoto(context: Context, uri: Uri, questionId: Int, skuId: String): String? {
        return try {
            val targetRelativePath = "visits/${questionId}/${skuId.ifEmpty { "gen" }}_gal_${System.currentTimeMillis()}.jpg"
            
             val tempCacheFile = File(context.cacheDir, "temp_gallery_import.jpg")
            if (tempCacheFile.exists()) {
                tempCacheFile.delete()
            }

            context.contentResolver.openInputStream(uri).use { inputStream ->
                if (inputStream == null) return null
                tempCacheFile.outputStream().use { outputStream ->
                    val buffer = ByteArray(4096)
                    var bytesRead: Int
                    while (inputStream.read(buffer).also { bytesRead = it } != -1) {
                        outputStream.write(buffer, 0, bytesRead)
                    }
                    outputStream.flush()
                }
            }

            if (!tempCacheFile.exists() || tempCacheFile.length() == 0L) return null

            val targetFile = File(context.filesDir, targetRelativePath)
            targetFile.parentFile?.mkdirs()

            val compressedFile = SecurityUtils.compressImageFile(
                context = context,
                sourceFile = tempCacheFile, 
                targetRelativePath = targetRelativePath 
            )

             if (tempCacheFile.exists()) {
                tempCacheFile.delete()
            }

            if (compressedFile != null && compressedFile.exists() && compressedFile.length() > 0) {
                targetRelativePath
            } else {
                if (targetFile.exists() && targetFile.length() > 0) targetRelativePath else null
            }
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }
}
