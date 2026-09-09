export const CONNECTION_LIMIT = 150;

export type Person = {
  id: string;
  displayName: string;
  username: string;
  initials: string;
};

export type ConnectionRequest = {
  id: string;
  person: Person;
};

export type PeopleSnapshot = {
  connections: Person[];
  incomingRequests: ConnectionRequest[];
};

export type ConnectionConflictCode =
  | "already_connected"
  | "already_pending"
  | "blocked"
  | "connection_limit_reached"
  | "not_found"
  | "request_resolved"
  | "self_request";

export class ConnectionConflict extends Error {
  constructor(public readonly code: ConnectionConflictCode) {
    super(code);
    this.name = "ConnectionConflict";
  }
}

export interface ConnectionsContract {
  listPeople(): Promise<PeopleSnapshot>;
  sendRequest(username: string): Promise<void>;
  acceptRequest(requestId: string): Promise<void>;
  rejectRequest(requestId: string): Promise<void>;
  removeConnection(personId: string): Promise<void>;
}

export function connectionErrorMessage(error: unknown): string {
  if (!(error instanceof ConnectionConflict)) {
    return "Something went wrong. Please try again.";
  }

  switch (error.code) {
    case "already_connected":
      return "You're already connected with this person.";
    case "already_pending":
      return "A request between you is already waiting for a response.";
    case "blocked":
      return "This connection request isn't available.";
    case "connection_limit_reached":
      return "This request can't be accepted because one of you already has 150 connections.";
    case "not_found":
      return "We couldn't find that username.";
    case "request_resolved":
      return "This request has already been answered. Refreshing People may help.";
    case "self_request":
      return "You can't send a connection request to yourself.";
  }
}
