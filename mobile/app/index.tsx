import { StatusBar } from "expo-status-bar";
import { StyleSheet, Text, View } from "react-native";

export default function WelcomeScreen() {
  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      <Text style={styles.title}>Tribe</Text>
      <Text style={styles.message}>A private place for your people.</Text>
      <Text style={styles.detail}>The mobile foundation is ready for the MVP.</Text>
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
  title: {
    color: "#2C2925",
    fontSize: 36,
    fontWeight: "600",
    marginBottom: 12
  },
  message: {
    color: "#4D4841",
    fontSize: 18,
    textAlign: "center"
  },
  detail: {
    color: "#756E65",
    fontSize: 15,
    marginTop: 24,
    textAlign: "center"
  }
});
