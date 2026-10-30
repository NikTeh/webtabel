package com.mg.sfa.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import android.util.Base64
import com.mg.sfa.data.local.AppDao
import com.mg.sfa.data.local.UserEntity
import com.mg.sfa.data.remote.ApiService
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoginScreen(
    apiService: ApiService,
    dao: AppDao, // ИСПРАВЛЕНО: Принимаем DAO локальной базы данных
    onLoginSuccess: (phone: String, pass: String, name: String) -> Unit
) {
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf("") }
    var isLoading by remember { mutableStateOf(false) }
    
    val scope = rememberCoroutineScope()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = "webtabel мерчандайзинг",
            fontSize = 24.sp,
            style = MaterialTheme.typography.headlineMedium,
            modifier = Modifier.padding(bottom = 32.dp)
        )

        if (errorMessage.isNotEmpty()) {
            Text(
                text = errorMessage,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.padding(bottom = 16.dp)
            )
        }
        
        OutlinedTextField(
            value = phone,
            onValueChange = { phone = it },
            label = { Text("Телефон (Логин)") },
            placeholder = { Text("79991112233") },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            singleLine = true,
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 16.dp)
        )

        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            label = { Text("Пароль") },
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
            singleLine = true,
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 24.dp)
        )

        Button(
            onClick = {
                errorMessage = ""
                val cleanPhone = phone.trim()
                if (cleanPhone.isEmpty() || password.isEmpty()) {
                    errorMessage = "Заполните все обязательные поля"
                    return@Button
                }

                isLoading = true
                
                val rawCredentials = "$cleanPhone:$password"
                val base64Header = "Basic " + Base64.encodeToString(
                    rawCredentials.toByteArray(), 
                    Base64.NO_WRAP
                )

                scope.launch {
                    try {
                        
                        val response = apiService.login(base64Header)
                        
                        if (response.role == "merchandiser") {
                            
                            dao.saveLocalUser(
                                UserEntity(
                                    phone = cleanPhone,
                                    passwordHash = password, 
                                    fullName = response.name,
                                    role = response.role
                                )
                            )
                            isLoading = false
                            onLoginSuccess(cleanPhone, password, response.name)
                        } else {
                            isLoading = false
                            errorMessage = "Доступ запрещен. Только для мерчандайзеров"
                        }
                    } catch (e: retrofit2.HttpException) {
                        isLoading = false
                        if (e.code() == 401) {
                            errorMessage = "Неверный номер телефона или пароль"
                        } else {
                            errorMessage = "Ошибка сервера: ${e.code()}"
                        }
                    } catch (e: java.io.IOException) {
                        
                        val cachedUser = dao.getLocalUser(cleanPhone)
                        
                        isLoading = false
                        if (cachedUser != null) {
                            // Сверяем введенный пароль с тем, что лежит в памяти SQLite телефона
                            if (cachedUser.passwordHash == password) {
                                // Пароль совпал! Пропускаем мерчандайзера в оффлайн-режиме
                                onLoginSuccess(cleanPhone, password, cachedUser.fullName)
                            } else {
                                errorMessage = "Неверный пароль для оффлайн-входа"
                            }
                        } else {
                            errorMessage = "Нет интернета. Оффлайн-вход невозможен, так как вы еще не авторизовывались на этом устройстве с интернетом"
                        }
                    } catch (e: Exception) {
                        isLoading = false
                        errorMessage = "Неизвестная ошибка: ${e.localizedMessage}"
                    }
                }
            },
            enabled = !isLoading,
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp)
        ) {
            if (isLoading) {
                CircularProgressIndicator(
                    color = MaterialTheme.colorScheme.onPrimary,
                    modifier = Modifier.size(24.dp)
                )
            } else {
                Text(text = "Войти", fontSize = 16.sp)
            }
        }
    }
}
