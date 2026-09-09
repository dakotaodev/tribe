import { useEffect } from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { useRouter } from "expo-router";

import { useSession } from "../../providers/session-provider";

export default function AuthCallbackScreen() {
  const { isLoading } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading) {
      router.replace("/");
    }
  }, [isLoading, router]);

  return (
    <View style={styles.container}>
      <ActivityIndicator color="#4D4841" />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: "center",
    backgroundColor: "#FAF8F4",
    flex: 1,
    justifyContent: "center"
  }
});
