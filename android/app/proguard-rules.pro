# 1. Защита Room СУБД и локальных сущностей Николая от краша при чтении диска
-keepclassmembers class * {
    @androidx.room.Entity *;
    @androidx.room.Dao *;
    @androidx.room.Database *;
    @androidx.room.PrimaryKey *;
}
-keep class com.mg.sfa.data.local.** { *; }

# 2. Защита сетевых структур данных Retrofit и Gson для обмена с webtabel.ru
-keep class com.mg.sfa.data.remote.** { *; }
-keepattributes Signature, InnerClasses, EnclosingMethod, AnnotationDefault
-dontwarn retrofit2.**
-keep class retrofit2.** { *; }

# 3. Сохраняем номера строк в логах терминала VS Code для быстрой отладки
-keepattributes SourceFile, LineNumberTable
