package com.example.noteapp

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.example.noteapp.data.AuthRepository
import com.example.noteapp.data.GraphQLClient
import com.example.noteapp.data.ListRepository
import com.example.noteapp.data.TokenStore
import com.example.noteapp.ui.auth.AuthViewModel
import com.example.noteapp.ui.auth.LoginScreen
import com.example.noteapp.ui.auth.RegisterScreen
import com.example.noteapp.ui.create.CreateListScreen
import com.example.noteapp.ui.lists.ListDetailScreen
import com.example.noteapp.ui.lists.ListDetailViewModel
import com.example.noteapp.ui.lists.ListsScreen
import com.example.noteapp.ui.lists.ListsViewModel
import com.example.noteapp.ui.navigation.Routes
import com.example.noteapp.ui.theme.NoteAppTheme
import kotlinx.coroutines.launch

// Change this to your machine's IP when testing on a physical device
private const val BASE_URL = "http://10.0.2.2:8080"

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        val tokenStore = TokenStore(applicationContext)
        val graphQLClient = GraphQLClient(BASE_URL, tokenStore)
        val authRepo = AuthRepository(graphQLClient, tokenStore)
        val listRepo = ListRepository(graphQLClient)

        setContent {
            NoteAppTheme {
                val navController = rememberNavController()
                val scope = rememberCoroutineScope()

                val authViewModel = remember { AuthViewModel(authRepo) }
                val listsViewModel = remember { ListsViewModel(listRepo) }
                val listDetailViewModel = remember { ListDetailViewModel(listRepo) }

                NavHost(navController = navController, startDestination = Routes.LOGIN) {
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
                            onLogout = {
                                authViewModel.logout {
                                    navController.navigate(Routes.LOGIN) {
                                        popUpTo(0) { inclusive = true }
                                    }
                                }
                            }
                        )
                    }

                    composable(Routes.LIST_DETAIL) { backStackEntry ->
                        val listId = backStackEntry.arguments?.getString("listId") ?: return@composable
                        ListDetailScreen(
                            listId = listId,
                            viewModel = listDetailViewModel,
                            onBack = { navController.popBackStack() }
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
                }
            }
        }
    }
}
