package com.example.noteapp.ui.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.noteapp.ui.theme.LocalAppColors
import com.example.noteapp.ui.theme.LocalStrings
import com.example.noteapp.ui.theme.NeuButton
import com.example.noteapp.ui.theme.NeuTextField

@Composable
fun LoginScreen(
    viewModel: AuthViewModel,
    onLoginSuccess: () -> Unit,
    onNavigateToRegister: () -> Unit,
) {
    val colors = LocalAppColors.current
    val strings = LocalStrings.current
    val state by viewModel.state.collectAsState()
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(colors.bg),
    ) {
        // Hero zone (top 40%)
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .fillMaxSize(0.42f)
                .background(colors.bg),
            contentAlignment = Alignment.Center,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Icon(
                    Icons.Default.Lock,
                    contentDescription = null,
                    tint = colors.accent,
                    modifier = Modifier.size(56.dp),
                )
                Spacer(Modifier.height(16.dp))
                Text(
                    strings.notesManager,
                    style = TextStyle(
                        color = colors.textPrimary,
                        fontWeight = FontWeight.Bold,
                        fontSize = 28.sp,
                    ),
                )
                Spacer(Modifier.height(6.dp))
                Text(
                    strings.signInToContinue,
                    style = TextStyle(color = colors.textSecondary, fontSize = 14.sp),
                )
            }
        }

        // Form card (bottom portion)
        Column(
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .background(colors.bg)
                .padding(horizontal = 16.dp, vertical = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                strings.welcomeBack,
                style = TextStyle(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.Bold,
                    fontSize = 24.sp,
                ),
            )
            Spacer(Modifier.height(24.dp))

            NeuTextField(
                value = email,
                onValueChange = { email = it },
                label = strings.email,
                leadingIcon = {
                    Icon(Icons.Default.Email, contentDescription = null, tint = colors.textSecondary, modifier = Modifier.size(20.dp))
                },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))

            NeuTextField(
                value = password,
                onValueChange = { password = it },
                label = strings.password,
                leadingIcon = {
                    Icon(Icons.Default.Lock, contentDescription = null, tint = colors.textSecondary, modifier = Modifier.size(20.dp))
                },
                trailingIcon = {
                    Icon(
                        if (passwordVisible) Icons.Default.VisibilityOff else Icons.Default.Visibility,
                        contentDescription = if (passwordVisible) strings.hidePassword else strings.showPassword,
                        tint = colors.textSecondary,
                        modifier = Modifier
                            .size(20.dp)
                            .clickable { passwordVisible = !passwordVisible },
                    )
                },
                visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))

            if (state.error != null) {
                Text(
                    state.error!!,
                    color = colors.error,
                    style = TextStyle(fontSize = 12.sp),
                    modifier = Modifier.padding(horizontal = 8.dp),
                )
                Spacer(Modifier.height(8.dp))
            }

            NeuButton(
                text = strings.signIn,
                onClick = { viewModel.login(email, password, onLoginSuccess) },
                enabled = !state.isLoading && email.isNotBlank() && password.isNotBlank(),
                loading = state.isLoading,
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(Modifier.height(8.dp))
            Text(
                strings.dontHaveAccount,
                color = colors.accent,
                style = TextStyle(fontSize = 14.sp),
                modifier = Modifier
                    .clickable(onClick = onNavigateToRegister)
                    .padding(8.dp),
            )
        }
    }
}
