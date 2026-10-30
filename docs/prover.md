PS C:\Users\Nikolay\Desktop\webtabel\android> ./gradlew clean assembleDebug

C:\Users\Nikolay\Desktop\webtabel\android>if "Windows_NT" == "Windows_NT" setlocal

C:\Users\Nikolay\Desktop\webtabel\android>set DIRNAME=C:\Users\Nikolay\Desktop\webtabel\android\ 

C:\Users\Nikolay\Desktop\webtabel\android>if "C:\Users\Nikolay\Desktop\webtabel\android\" == "" set DIRNAME=. 

C:\Users\Nikolay\Desktop\webtabel\android>set APP_BASE_NAME=gradlew 

C:\Users\Nikolay\Desktop\webtabel\android>set APP_HOME=C:\Users\Nikolay\Desktop\webtabel\android\ 

C:\Users\Nikolay\Desktop\webtabel\android>for %i in ("C:\Users\Nikolay\Desktop\webtabel\android\") do set APP_HOME=%~fi 

C:\Users\Nikolay\Desktop\webtabel\android>set APP_HOME=C:\Users\Nikolay\Desktop\webtabel\android\ 

C:\Users\Nikolay\Desktop\webtabel\android>if defined JAVA_HOME goto findJavaFromJavaHome 

C:\Users\Nikolay\Desktop\webtabel\android>set JAVA_HOME=C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\ 

C:\Users\Nikolay\Desktop\webtabel\android>set JAVA_EXE=C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\\bin\java.exe 

C:\Users\Nikolay\Desktop\webtabel\android>if exist "C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\\bin\java.exe" goto execute 

C:\Users\Nikolay\Desktop\webtabel\android>set CLASSPATH=C:\Users\Nikolay\Desktop\webtabel\android\\gradle\wrapper\gradle-wrapper.jar 

C:\Users\Nikolay\Desktop\webtabel\android>"C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\\bin\java.exe" "-Dorg.gradle.appname=gradlew" -classpath "C:\Users\Nikolay\Desktop\webtabel\android\\gradle\wrapper\gradle-wrapper.jar" org.gradle.wrapper.GradleWrapperMain clean assembleDebug 

> Task :app:stripDebugDebugSymbols
Unable to strip the following libraries, packaging them as they are: libimage_processing_util_jni.so.

> Task :app:compileDebugKotlin
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/data/repository/DataRepository.kt:197:68 'create(MediaType?, String): RequestBody' is deprecated. Moved to extension function. Put the 'content' argument first to fix Java
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/data/repository/DataRepository.kt:198:71 'create(MediaType?, File): RequestBody' is deprecated. Moved to extension function. Put the 'file' argument first to fix Java
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/data/repository/DataRepository.kt:201:68 'create(MediaType?, String): RequestBody' is deprecated. Moved to extension function. Put the 'content' argument first to fix Java
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/CameraCaptureScreen.kt:41:9 Variable 'cameraExecutor' is never used
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/RoutesScreen.kt:177:29 Name shadowed: context
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/RoutesScreen.kt:178:29 Name shadowed: dao
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/StoresScreen.kt:93:29 Name shadowed: context
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/StoresScreen.kt:94:29 Name shadowed: dao
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/util/SecurityUtils.kt:54:25 'getter for isFromMockProvider: Boolean' is deprecated. Deprecated in Java
The Daemon will expire after the build after running out of JVM heap space.
The project memory settings are likely not configured or are configured to an insufficient value.
The daemon will restart for the next build, which may increase subsequent build times.
These settings can be adjusted by setting 'org.gradle.jvmargs' in 'gradle.properties'.
The currently configured max heap space is '512 MiB' and the configured max metaspace is '384 MiB'.
For more information on how to set these values, visit the user guide at https://docs.gradle.org/8.1/userguide/build_environment.html#sec:configuring_jvm_memory
To disable this warning, set 'org.gradle.daemon.performance.disable-logging=true'.
Daemon will be stopped at the end of the build after running out of JVM heap space

> Task :app:minifyDebugWithR8 FAILED
ERROR: R8: java.lang.OutOfMemoryError: Java heap space

FAILURE: Build failed with an exception.

* What went wrong:
Execution failed for task ':app:minifyDebugWithR8'.
> A failure occurred while executing com.android.build.gradle.internal.tasks.R8Task$R8Runnable
   > Compilation failed to complete

* Try:
> Run with --stacktrace option to get the stack trace.
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Get more help at https://help.gradle.org

BUILD FAILED in 2m 51s
35 actionable tasks: 35 executed

C:\Users\Nikolay\Desktop\webtabel\android>if "1" == "0" goto mainEnd 

C:\Users\Nikolay\Desktop\webtabel\android>exit /b 1 
PS C:\Users\Nikolay\Desktop\webtabel\android> ./gradlew clean assembleDebug

C:\Users\Nikolay\Desktop\webtabel\android>if "Windows_NT" == "Windows_NT" setlocal

C:\Users\Nikolay\Desktop\webtabel\android>set DIRNAME=C:\Users\Nikolay\Desktop\webtabel\android\ 

C:\Users\Nikolay\Desktop\webtabel\android>if "C:\Users\Nikolay\Desktop\webtabel\android\" == "" set DIRNAME=. 

C:\Users\Nikolay\Desktop\webtabel\android>set APP_BASE_NAME=gradlew 

C:\Users\Nikolay\Desktop\webtabel\android>set APP_HOME=C:\Users\Nikolay\Desktop\webtabel\android\ 

C:\Users\Nikolay\Desktop\webtabel\android>for %i in ("C:\Users\Nikolay\Desktop\webtabel\android\") do set APP_HOME=%~fi 

C:\Users\Nikolay\Desktop\webtabel\android>set APP_HOME=C:\Users\Nikolay\Desktop\webtabel\android\ 

C:\Users\Nikolay\Desktop\webtabel\android>if defined JAVA_HOME goto findJavaFromJavaHome 

C:\Users\Nikolay\Desktop\webtabel\android>set JAVA_HOME=C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\ 

C:\Users\Nikolay\Desktop\webtabel\android>set JAVA_EXE=C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\\bin\java.exe 

C:\Users\Nikolay\Desktop\webtabel\android>if exist "C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\\bin\java.exe" goto execute 

C:\Users\Nikolay\Desktop\webtabel\android>set CLASSPATH=C:\Users\Nikolay\Desktop\webtabel\android\\gradle\wrapper\gradle-wrapper.jar 

C:\Users\Nikolay\Desktop\webtabel\android>"C:\Program Files\Eclipse Adoptium\jdk-17.0.20.8-hotspot\\bin\java.exe" "-Dorg.gradle.appname=gradlew" -classpath "C:\Users\Nikolay\Desktop\webtabel\android\\gradle\wrapper\gradle-wrapper.jar" org.gradle.wrapper.GradleWrapperMain clean assembleDebug 
Starting a Gradle Daemon, 1 stopped Daemon could not be reused, use --status for details
Warning: SDK processing. This version only understands SDK XML versions up to 3 but an SDK XML file of version 4 was encountered. This can happen if you use versions of Android Studio and the command-line tools that were released at different times.

> Task :app:stripDebugDebugSymbols
Unable to strip the following libraries, packaging them as they are: libimage_processing_util_jni.so.

> Task :app:compileDebugKotlin
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/data/repository/DataRepository.kt:197:68 'create(MediaType?, String): RequestBody' is deprecated. Moved to extension function. Put the 'content' argument first to fix Java
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/data/repository/DataRepository.kt:198:71 'create(MediaType?, File): RequestBody' is deprecated. Moved to extension function. Put the 'file' argument first to fix Java
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/data/repository/DataRepository.kt:201:68 'create(MediaType?, String): RequestBody' is deprecated. Moved to extension function. Put the 'content' argument first to fix Java
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/CameraCaptureScreen.kt:41:9 Variable 'cameraExecutor' is never used
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/RoutesScreen.kt:177:29 Name shadowed: context
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/RoutesScreen.kt:178:29 Name shadowed: dao
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/StoresScreen.kt:93:29 Name shadowed: context
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/ui/screens/StoresScreen.kt:94:29 Name shadowed: dao
w: file:///C:/Users/Nikolay/Desktop/webtabel/android/app/src/main/java/com/mg/sfa/util/SecurityUtils.kt:54:25 'getter for isFromMockProvider: Boolean' is deprecated. Deprecated in Java
The Daemon will expire after the build after running out of JVM heap space.
The project memory settings are likely not configured or are configured to an insufficient value.
The daemon will restart for the next build, which may increase subsequent build times.
These settings can be adjusted by setting 'org.gradle.jvmargs' in 'gradle.properties'.
The currently configured max heap space is '512 MiB' and the configured max metaspace is '384 MiB'.
For more information on how to set these values, visit the user guide at https://docs.gradle.org/8.1/userguide/build_environment.html#sec:configuring_jvm_memory
To disable this warning, set 'org.gradle.daemon.performance.disable-logging=true'.
Daemon will be stopped at the end of the build after running out of JVM heap space

FAILURE: Build failed with an exception.

* What went wrong:
Could not receive a message from the daemon.

* Try:
> Run with --stacktrace option to get the stack trace.
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Get more help at https://help.gradle.org

C:\Users\Nikolay\Desktop\webtabel\android>if "1" == "0" goto mainEnd 

C:\Users\Nikolay\Desktop\webtabel\android>exit /b 1 
PS C:\Users\Nikolay\Desktop\webtabel\android> 