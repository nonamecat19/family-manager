package com.example.noteapp

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.example.noteapp.data.AuthRepository
import com.example.noteapp.data.GraphQLClient
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.SettingsRepository
import com.example.noteapp.data.TokenStore
import com.example.noteapp.ui.auth.AuthViewModel
import com.example.noteapp.ui.auth.LoginScreen
import com.example.noteapp.ui.auth.RegisterScreen
import com.example.noteapp.ui.create.CreateListScreen
import com.example.noteapp.ui.lists.EditListScreen
import com.example.noteapp.ui.lists.ListDetailScreen
import com.example.noteapp.ui.lists.ListDetailViewModel
import com.example.noteapp.ui.lists.ListsScreen
import com.example.noteapp.ui.lists.ListsViewModel
import com.example.noteapp.ui.navigation.Routes
import com.example.noteapp.ui.settings.SettingsScreen
import com.example.noteapp.ui.settings.SettingsViewModel
import com.example.noteapp.ui.theme.AppColors
import com.example.noteapp.ui.theme.AppTheme
import com.example.noteapp.ui.theme.EnglishStrings
import com.example.noteapp.ui.theme.UkrainianStrings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.firstOrNull
import kotlinx.coroutines.launch

private const val BASE_URL = "https://family-manager-pwll.onrender.com"

class MainActivity : ComponentActivity() {
    private val pendingListId = MutableStateFlow<String?>(null)

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        pendingListId.value = intent.getStringExtra("openListId")
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        pendingListId.value = intent.getStringExtra("openListId")
        enableEdgeToEdge()

        val tokenStore = TokenStore(applicationContext)
        val graphQLClient = GraphQLClient(BASE_URL, tokenStore)
        val authRepo = AuthRepository(graphQLClient, tokenStore)
        val listRepo = ListRepository(graphQLClient)
        val settingsRepo = SettingsRepository(applicationContext)

        setContent {
            val settingsViewModel = remember { SettingsViewModel(settingsRepo) }
            val language by settingsViewModel.language.collectAsState()
            val accentColor by settingsViewModel.accentColor.collectAsState()

            val appColors = AppColors(accent = accentColor, accentDim = accentColor.copy(alpha = 0.1f))
            val appStrings = if (language == "uk") UkrainianStrings else EnglishStrings

            AppTheme(colors = appColors, strings = appStrings) {
                val navController = rememberNavController()
                val scope = rememberCoroutineScope()
                val openListId by pendingListId.collectAsState()

                val authViewModel = remember { AuthViewModel(authRepo) }
                val listsViewModel = remember { ListsViewModel(listRepo) }
                val listDetailViewModel = remember { ListDetailViewModel(listRepo) }

                // Handle widget click when app is already running (onNewIntent case)
                LaunchedEffect(openListId) {
                    val listId = openListId ?: return@LaunchedEffect
                    val currentRoute = navController.currentBackStackEntry?.destination?.route
                    if (currentRoute != null && currentRoute != Routes.SPLASH) {
                        pendingListId.value = null
                        navController.navigate(Routes.listDetail(listId))
                    }
                }

                NavHost(
                    navController = navController,
                    startDestination = Routes.SPLASH,
                    enterTransition = { EnterTransition.None },
                    exitTransition = { ExitTransition.None },
                    popEnterTransition = { EnterTransition.None },
                    popExitTransition = { ExitTransition.None },
                ) {

                    composable(Routes.SPLASH) {
                        Box(
                            Modifier
                                .fillMaxSize()
                                .background(appColors.bg),
                            contentAlignment = Alignment.Center,
                        ) {
                            CircularProgressIndicator(color = appColors.accent)
                        }
                        LaunchedEffect(Unit) {
                            val isAuthed = try {
                                val token = tokenStore.accessToken.firstOrNull()
                                if (token != null) {
                                    try { authRepo.me(); true }
                                    catch (_: Exception) {
                                        try { authRepo.refresh(); true }
                                        catch (_: Exception) { false }
                                    }
                                } else false
                            } catch (_: Exception) { false }

                            val destination = if (isAuthed) {
                                val listId = pendingListId.value?.also { pendingListId.value = null }
                                if (listId != null) Routes.listDetail(listId) else Routes.LISTS
                            } else Routes.LOGIN

                            navController.navigate(destination) {
                                popUpTo(Routes.SPLASH) { inclusive = true }
                            }
                        }
                    }

                    composable(Routes.LOGIN) {
                        LoginScreen(
                            viewModel = authViewModel,
                            onLoginSuccess = {
                                navController.navigate(Routes.LISTS) {
                                    popUpTo(Routes.LOGIN) { inclusive = true }
                                }
                            },
                            onNavigateToRegister = {
                                navController.navigate(Routes.REGISTER)
                            }
                        )
                    }

                    composable(Routes.REGISTER) {
                        RegisterScreen(
                            viewModel = authViewModel,
                            onRegisterSuccess = {
                                navController.navigate(Routes.LISTS) {
                                    popUpTo(Routes.LOGIN) { inclusive = true }
                                }
                            },
                            onNavigateToLogin = { navController.popBackStack() }
                        )
                    }

                    composable(Routes.LISTS) {
                        ListsScreen(
                            viewModel = listsViewModel,
                            onListClick = { id ->
                                navController.navigate(Routes.listDetail(id))
                            },
                            onCreateList = {
                                navController.navigate(Routes.CREATE_LIST)
                            },
                            onSettings = {
                                navController.navigate(Routes.SETTINGS)
                            }
                        )
                    }

                    composable(Routes.LIST_DETAIL) { backStackEntry ->
                        val listId = backStackEntry.arguments?.getString("listId") ?: return@composable
                        ListDetailScreen(
                            listId = listId,
                            viewModel = listDetailViewModel,
                            onBack = { navController.popBackStack() },
                            onEditList = { id -> navController.navigate(Routes.editList(id)) },
                        )
                    }

                    composable(Routes.EDIT_LIST) { backStackEntry ->
                        val listId = backStackEntry.arguments?.getString("listId") ?: return@composable
                        EditListScreen(
                            listId = listId,
                            viewModel = listDetailViewModel,
                            onBack = { navController.popBackStack() },
                        )
                    }

                    composable(Routes.CREATE_LIST) {
                        CreateListScreen(
                            onBack = { navController.popBackStack() },
                            onCreate = { title, description, isPublic ->
                                scope.launch {
                                    try {
                                        listRepo.createList(title, description, isPublic)
                                        navController.popBackStack()
                                        listsViewModel.loadMyLists()
                                    } catch (_: Exception) {}
                                }
                            }
                        )
                    }

                    composable(Routes.SETTINGS) {
                        SettingsScreen(
                            viewModel = settingsViewModel,
                            onBack = { navController.popBackStack() },
                            onLogout = {
                                authViewModel.logout {
                                    navController.navigate(Routes.LOGIN) {
                                        popUpTo(0) { inclusive = true }
                                    }
                                }
                            }
                        )
                    }
                }
            }
        }
    }
}
