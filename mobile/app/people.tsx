import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";
import {
  CONNECTION_LIMIT,
  connectionErrorMessage,
  type ConnectionRequest,
  type PeopleSnapshot,
  type Person
} from "../src/connections/contract";
import { mockConnections } from "../src/connections/mock";

type Action = { kind: "accept" | "reject" | "remove"; id: string } | null;

export default function PeopleScreen() {
  const [people, setPeople] = useState<PeopleSnapshot | null>(null);
  const [loadError, setLoadError] = useState("");
  const [notice, setNotice] = useState("");
  const [username, setUsername] = useState("");
  const [sending, setSending] = useState(false);
  const [action, setAction] = useState<Action>(null);

  const loadPeople = useCallback(async () => {
    setLoadError("");
    try {
      setPeople(await mockConnections.listPeople());
    } catch {
      setLoadError("We couldn't load your people. Check your connection and try again.");
    }
  }, []);

  useEffect(() => {
    void loadPeople();
  }, [loadPeople]);

  const runRequestAction = async (
    kind: "accept" | "reject",
    request: ConnectionRequest
  ) => {
    setNotice("");
    setAction({ kind, id: request.id });
    try {
      if (kind === "accept") await mockConnections.acceptRequest(request.id);
      else await mockConnections.rejectRequest(request.id);
      await loadPeople();
      setNotice(
        kind === "accept"
          ? `You and ${request.person.displayName} are now connected.`
          : `Request from ${request.person.displayName} declined.`
      );
    } catch (error) {
      setNotice(connectionErrorMessage(error));
    } finally {
      setAction(null);
    }
  };

  const sendRequest = async () => {
    if (!username.trim()) return;
    setSending(true);
    setNotice("");
    try {
      await mockConnections.sendRequest(username);
      setNotice(`Request sent to @${username.trim().replace(/^@/, "")}.`);
      setUsername("");
    } catch (error) {
      setNotice(connectionErrorMessage(error));
    } finally {
      setSending(false);
    }
  };

  const confirmRemove = (person: Person) => {
    Alert.alert(
      `Remove ${person.displayName}?`,
      "You'll no longer be connected. Either of you can send a new request later.",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Remove",
          style: "destructive",
          onPress: () => {
            setAction({ kind: "remove", id: person.id });
            setNotice("");
            void mockConnections
              .removeConnection(person.id)
              .then(loadPeople)
              .then(() => setNotice(`${person.displayName} was removed from your people.`))
              .catch((error) => setNotice(connectionErrorMessage(error)))
              .finally(() => setAction(null));
          }
        }
      ]
    );
  };

  return (
    <SafeAreaView style={styles.safeArea}>
      <StatusBar style="dark" />
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <View style={styles.headingRow}>
          <View>
            <Text style={styles.eyebrow}>YOUR TRIBE</Text>
            <Text style={styles.title}>People</Text>
          </View>
          <View style={styles.countPill} accessibilityLabel={`${people?.connections.length ?? 0} of ${CONNECTION_LIMIT} connections`}>
            <Text style={styles.count}>{people?.connections.length ?? "—"} / {CONNECTION_LIMIT}</Text>
          </View>
        </View>

        <Text style={styles.intro}>A quiet place for the people you know. Connections are always mutual.</Text>

        <View style={styles.inviteCard}>
          <Text style={styles.sectionTitle}>Connect by username</Text>
          <Text style={styles.helper}>Send a request to someone you already know.</Text>
          <View style={styles.inputRow}>
            <TextInput
              accessibilityLabel="Username"
              autoCapitalize="none"
              autoCorrect={false}
              onChangeText={setUsername}
              onSubmitEditing={() => void sendRequest()}
              placeholder="@username"
              placeholderTextColor="#948C81"
              returnKeyType="send"
              style={styles.input}
              value={username}
            />
            <Pressable
              accessibilityRole="button"
              disabled={!username.trim() || sending}
              onPress={() => void sendRequest()}
              style={({ pressed }) => [styles.sendButton, (!username.trim() || sending) && styles.disabled, pressed && styles.pressed]}
            >
              {sending ? <ActivityIndicator color="#FFFFFF" /> : <Text style={styles.sendText}>Send</Text>}
            </Pressable>
          </View>
        </View>

        {!!notice && <Text accessibilityLiveRegion="polite" style={styles.notice}>{notice}</Text>}

        {!people && !loadError && <LoadingState />}
        {!!loadError && (
          <View style={styles.stateCard}>
            <Text style={styles.stateTitle}>People aren't available</Text>
            <Text style={styles.stateText}>{loadError}</Text>
            <Pressable accessibilityRole="button" onPress={() => void loadPeople()} style={styles.retryButton}>
              <Text style={styles.retryText}>Try again</Text>
            </Pressable>
          </View>
        )}

        {people && (
          <>
            <SectionHeader title="Requests" count={people.incomingRequests.length} />
            {people.incomingRequests.length === 0 ? (
              <EmptyState title="No requests waiting" message="New connection requests will appear here." />
            ) : (
              people.incomingRequests.map((request) => (
                <PersonRow key={request.id} person={request.person}>
                  <Pressable
                    accessibilityLabel={`Decline request from ${request.person.displayName}`}
                    disabled={action !== null}
                    onPress={() => void runRequestAction("reject", request)}
                    style={({ pressed }) => [styles.secondaryButton, pressed && styles.pressed]}
                  >
                    <Text style={styles.secondaryText}>{action?.kind === "reject" && action.id === request.id ? "…" : "Decline"}</Text>
                  </Pressable>
                  <Pressable
                    accessibilityLabel={`Accept request from ${request.person.displayName}`}
                    disabled={action !== null}
                    onPress={() => void runRequestAction("accept", request)}
                    style={({ pressed }) => [styles.acceptButton, pressed && styles.pressed]}
                  >
                    <Text style={styles.acceptText}>{action?.kind === "accept" && action.id === request.id ? "…" : "Accept"}</Text>
                  </Pressable>
                </PersonRow>
              ))
            )}

            <SectionHeader title="Connections" count={people.connections.length} />
            {people.connections.length === 0 ? (
              <EmptyState title="Your people will show up here" message="Connect by username to begin your Tribe." />
            ) : (
              people.connections.map((person) => (
                <PersonRow key={person.id} person={person}>
                  <Pressable
                    accessibilityLabel={`Remove ${person.displayName}`}
                    disabled={action !== null}
                    onPress={() => confirmRemove(person)}
                    hitSlop={8}
                  >
                    <Text style={styles.removeText}>{action?.kind === "remove" && action.id === person.id ? "Removing…" : "Remove"}</Text>
                  </Pressable>
                </PersonRow>
              ))
            )}
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

function SectionHeader({ title, count }: { title: string; count: number }) {
  return <View style={styles.sectionHeader}><Text style={styles.sectionTitle}>{title}</Text><Text style={styles.sectionCount}>{count}</Text></View>;
}

function PersonRow({ person, children }: { person: Person; children: React.ReactNode }) {
  return (
    <View style={styles.personRow}>
      <View style={styles.avatar}><Text style={styles.avatarText}>{person.initials}</Text></View>
      <View style={styles.personInfo}><Text style={styles.personName}>{person.displayName}</Text><Text style={styles.username}>@{person.username}</Text></View>
      <View style={styles.actions}>{children}</View>
    </View>
  );
}

function LoadingState() {
  return <View style={styles.loading}><ActivityIndicator color="#6D5540" /><Text style={styles.stateText}>Gathering your people…</Text></View>;
}

function EmptyState({ title, message }: { title: string; message: string }) {
  return <View style={styles.empty}><Text style={styles.stateTitle}>{title}</Text><Text style={styles.stateText}>{message}</Text></View>;
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: "#FAF8F4", flex: 1 },
  content: { paddingBottom: 48, paddingHorizontal: 20, paddingTop: 24 },
  headingRow: { alignItems: "center", flexDirection: "row", justifyContent: "space-between" },
  eyebrow: { color: "#8B6F58", fontSize: 11, fontWeight: "700", letterSpacing: 1.4, marginBottom: 4 },
  title: { color: "#292622", fontSize: 34, fontWeight: "700", letterSpacing: -0.8 },
  countPill: { backgroundColor: "#EEE7DE", borderRadius: 18, paddingHorizontal: 14, paddingVertical: 8 },
  count: { color: "#5B5148", fontSize: 14, fontVariant: ["tabular-nums"], fontWeight: "600" },
  intro: { color: "#716A62", fontSize: 15, lineHeight: 22, marginBottom: 22, marginTop: 10, maxWidth: 330 },
  inviteCard: { backgroundColor: "#FFFFFF", borderColor: "#E8E1D8", borderRadius: 18, borderWidth: 1, padding: 16 },
  sectionTitle: { color: "#332F2A", fontSize: 18, fontWeight: "600" },
  helper: { color: "#7B746C", fontSize: 13, marginTop: 5 },
  inputRow: { flexDirection: "row", gap: 10, marginTop: 14 },
  input: { backgroundColor: "#FAF8F4", borderColor: "#DDD5CA", borderRadius: 12, borderWidth: 1, color: "#292622", flex: 1, fontSize: 16, height: 46, paddingHorizontal: 13 },
  sendButton: { alignItems: "center", backgroundColor: "#634D3B", borderRadius: 12, justifyContent: "center", minWidth: 72 },
  sendText: { color: "#FFFFFF", fontSize: 15, fontWeight: "600" },
  disabled: { opacity: 0.45 },
  pressed: { opacity: 0.7 },
  notice: { backgroundColor: "#F1ECE5", borderRadius: 10, color: "#554B42", fontSize: 14, lineHeight: 20, marginTop: 14, padding: 12 },
  sectionHeader: { alignItems: "center", flexDirection: "row", gap: 8, marginBottom: 8, marginTop: 28 },
  sectionCount: { color: "#8A8177", fontSize: 14, fontWeight: "600" },
  personRow: { alignItems: "center", borderBottomColor: "#E9E3DB", borderBottomWidth: StyleSheet.hairlineWidth, flexDirection: "row", minHeight: 72, paddingVertical: 10 },
  avatar: { alignItems: "center", backgroundColor: "#E6DDD2", borderRadius: 22, height: 44, justifyContent: "center", width: 44 },
  avatarText: { color: "#625345", fontSize: 14, fontWeight: "700" },
  personInfo: { flex: 1, marginLeft: 12 },
  personName: { color: "#332F2A", fontSize: 16, fontWeight: "600" },
  username: { color: "#857D74", fontSize: 13, marginTop: 2 },
  actions: { alignItems: "center", flexDirection: "row", gap: 7 },
  secondaryButton: { borderColor: "#D8D0C6", borderRadius: 10, borderWidth: 1, paddingHorizontal: 10, paddingVertical: 8 },
  secondaryText: { color: "#625B54", fontSize: 13, fontWeight: "600" },
  acceptButton: { backgroundColor: "#634D3B", borderRadius: 10, paddingHorizontal: 11, paddingVertical: 9 },
  acceptText: { color: "#FFFFFF", fontSize: 13, fontWeight: "600" },
  removeText: { color: "#8B625B", fontSize: 13, fontWeight: "600" },
  loading: { alignItems: "center", gap: 12, paddingVertical: 64 },
  stateCard: { alignItems: "center", backgroundColor: "#FFFFFF", borderRadius: 16, marginTop: 24, padding: 24 },
  empty: { alignItems: "center", borderColor: "#E8E1D8", borderRadius: 14, borderStyle: "dashed", borderWidth: 1, padding: 22 },
  stateTitle: { color: "#4B453E", fontSize: 15, fontWeight: "600", textAlign: "center" },
  stateText: { color: "#817970", fontSize: 14, lineHeight: 20, marginTop: 5, textAlign: "center" },
  retryButton: { marginTop: 16, padding: 8 },
  retryText: { color: "#634D3B", fontSize: 15, fontWeight: "700" }
});
