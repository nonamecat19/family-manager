import { Stack } from 'expo-router';
import { ThemeProvider } from '@/theme/theme-provider';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';
import { FamilyProvider } from '@/contexts/FamilyContext';
import { NotificationProvider } from '@/contexts/NotificationContext';

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
          <Stack.Screen name='lists/create' options={{ title: 'Create List' }} />
          <Stack.Screen name='lists/[id]' options={{ title: 'List Details' }} />
          <Stack.Screen name='lists/[id]/edit' options={{ title: 'Edit List' }} />
          <Stack.Screen name='notes/create' options={{ title: 'Create Note' }} />
          <Stack.Screen name='notes/[id]' options={{ title: 'Note Details' }} />
          <Stack.Screen name='notes/[id]/edit' options={{ title: 'Edit Note' }} />
          <Stack.Screen name='birthdays/create' options={{ title: 'Add Birthday' }} />
          <Stack.Screen name='birthdays/[id]' options={{ title: 'Birthday Details' }} />
          <Stack.Screen name='birthdays/[id]/edit' options={{ title: 'Edit Birthday' }} />
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
          <NotificationProvider>
            <RootLayoutNav />
          </NotificationProvider>
        </FamilyProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

