import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { ThemeProvider } from '@/theme/theme-provider';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';
import { FamilyProvider } from '@/contexts/FamilyContext';
import { Redirect } from 'expo-router';

function RootLayoutNav() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return null; // Or a loading screen
  }

  return (
    <Stack>
      <Stack.Screen name='index' options={{ headerShown: false }} />
      {!isAuthenticated ? (
        <>
          <Stack.Screen name='auth/login' options={{ title: 'Login' }} />
          <Stack.Screen name='auth/register' options={{ title: 'Register' }} />
          <Stack.Screen name='(tabs)' options={{ headerShown: false }} />
        </>
      ) : (
        <>
          <Stack.Screen name='(tabs)' options={{ headerShown: false }} />
          <Stack.Screen name='auth/login' options={{ title: 'Login' }} />
          <Stack.Screen name='auth/register' options={{ title: 'Register' }} />
          <Stack.Screen name='families' options={{ title: 'Workspaces' }} />
          <Stack.Screen name='families/create' options={{ title: 'Create Workspace' }} />
        </>
      )}
      <Stack.Screen name='+not-found' />
    </Stack>
  );
}

export default function RootLayout() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <FamilyProvider>
          <RootLayoutNav />
          <StatusBar style='auto' />
        </FamilyProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

