import {
  CONNECTION_LIMIT,
  ConnectionConflict,
  type ConnectionsContract,
  type PeopleSnapshot
} from "./contract";

const wait = (milliseconds = 350) =>
  new Promise<void>((resolve) => setTimeout(resolve, milliseconds));

let snapshot: PeopleSnapshot = {
  connections: [
    { id: "maya", displayName: "Maya Chen", username: "mayac", initials: "MC" },
    { id: "jonah", displayName: "Jonah Bell", username: "jonahb", initials: "JB" },
    { id: "lena", displayName: "Lena Ortiz", username: "lenao", initials: "LO" }
  ],
  incomingRequests: [
    {
      id: "request-nora",
      person: { id: "nora", displayName: "Nora James", username: "noraj", initials: "NJ" }
    }
  ]
};

const knownPeople = {
  alexm: { id: "alex", displayName: "Alex Morgan", username: "alexm", initials: "AM" },
  blocked: { id: "blocked", displayName: "Unavailable", username: "blocked", initials: "" }
};

export const mockConnections: ConnectionsContract = {
  async listPeople() {
    await wait();
    return structuredClone(snapshot);
  },

  async sendRequest(username) {
    await wait();
    const normalized = username.trim().replace(/^@/, "").toLowerCase();
    if (normalized === "you") throw new ConnectionConflict("self_request");
    if (normalized === "blocked") throw new ConnectionConflict("blocked");
    if (snapshot.connections.some((person) => person.username === normalized)) {
      throw new ConnectionConflict("already_connected");
    }
    if (snapshot.incomingRequests.some(({ person }) => person.username === normalized)) {
      throw new ConnectionConflict("already_pending");
    }
    if (!(normalized in knownPeople)) throw new ConnectionConflict("not_found");
  },

  async acceptRequest(requestId) {
    await wait();
    const request = snapshot.incomingRequests.find(({ id }) => id === requestId);
    if (!request) throw new ConnectionConflict("request_resolved");
    if (snapshot.connections.length >= CONNECTION_LIMIT) {
      throw new ConnectionConflict("connection_limit_reached");
    }
    snapshot = {
      connections: [...snapshot.connections, request.person],
      incomingRequests: snapshot.incomingRequests.filter(({ id }) => id !== requestId)
    };
  },

  async rejectRequest(requestId) {
    await wait();
    if (!snapshot.incomingRequests.some(({ id }) => id === requestId)) {
      throw new ConnectionConflict("request_resolved");
    }
    snapshot = {
      ...snapshot,
      incomingRequests: snapshot.incomingRequests.filter(({ id }) => id !== requestId)
    };
  },

  async removeConnection(personId) {
    await wait();
    snapshot = {
      ...snapshot,
      connections: snapshot.connections.filter(({ id }) => id !== personId)
    };
  }
};
