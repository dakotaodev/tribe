import { StatusBar } from "expo-status-bar";
import { useState } from "react";
import { ActivityIndicator, Button, StyleSheet, Text, TextInput, View } from "react-native";

import { useSession } from "../providers/session-provider";

export default function WelcomeScreen() {
  const { error, isLoading, sendMagicLink, session, signInWithApple, signOut } = useSession();
  const [email, setEmail] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  const run = async (action: () => Promise<void>, successMessage?: string) => {
    setIsSubmitting(true);
    setMessage(null);
    try {
      await action();
      setMessage(successMessage ?? null);
    } catch {
      // The session provider exposes the actionable error state.
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      <Text style={styles.title}>Tribe</Text>
      {isLoading ? (
        <>
          <ActivityIndicator color="#4D4841" />
          <Text style={styles.message}>Restoring your session…</Text>
        </>
      ) : session ? (
        <>
          <Text style={styles.message}>You’re signed in.</Text>
          <Text style={styles.detail}>{session.user.email ?? "Apple account"}</Text>
          <Button disabled={isSubmitting} onPress={() => void run(signOut)} title="Sign out" />
        </>
      ) : (
        <View style={styles.form}>
          <Text style={styles.message}>A private place for your people.</Text>
          <Button disabled={isSubmitting} onPress={() => void run(signInWithApple)} title="Continue with Apple" />
          <Text style={styles.or}>or</Text>
          <TextInput
            accessibilityLabel="Email address"
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            onChangeText={setEmail}
            placeholder="you@example.com"
            style={styles.input}
            value={email}
          />
          <Button
            disabled={isSubmitting || email.trim().length === 0}
            onPress={() => void run(() => sendMagicLink(email.trim()), "Check your email for a sign-in link.")}
            title="Email me a sign-in link"
          />
        </View>
      )}
      {message ? <Text style={styles.success}>{message}</Text> : null}
      {error ? <Text accessibilityRole="alert" style={styles.error}>{error.message}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: "center",
    backgroundColor: "#FAF8F4",
    flex: 1,
    justifyContent: "center",
    padding: 24
  },
  detail: {
    color: "#756E65",
    fontSize: 15,
    marginBottom: 24,
    marginTop: 8,
    textAlign: "center"
  },
  error: {
    color: "#9A3D32",
    fontSize: 15,
    marginTop: 20,
    textAlign: "center"
  },
  form: {
    gap: 14,
    width: "100%"
  },
  input: {
    backgroundColor: "#FFFFFF",
    borderColor: "#D6D0C8",
    borderRadius: 8,
    borderWidth: 1,
    color: "#2C2925",
    fontSize: 16,
    padding: 12
  },
  message: {
    color: "#4D4841",
    fontSize: 18,
    marginBottom: 20,
    textAlign: "center"
  },
  or: {
    color: "#756E65",
    textAlign: "center"
  },
  success: {
    color: "#3C6A4A",
    fontSize: 15,
    marginTop: 20,
    textAlign: "center"
  },
  title: {
    color: "#2C2925",
    fontSize: 36,
    fontWeight: "600",
    marginBottom: 12
  }
});
