import { Tabs } from 'expo-router';

export default function TabsLayout() {
  return (
    <Tabs>
      <Tabs.Screen
        name="index"
        options={{
          title: 'Home',
        }}
      />
      <Tabs.Screen
        name="lists"
        options={{
          title: 'Lists',
        }}
      />
      <Tabs.Screen
        name="notes"
        options={{
          title: 'Notes',
        }}
      />
      <Tabs.Screen
        name="birthdays"
        options={{
          title: 'Birthdays',
        }}
      />
      <Tabs.Screen
        name="settings"
        options={{
          title: 'Settings',
        }}
      />
    </Tabs>
  );
}

