.. _ref-task-sendmessage:

SENDMESSAGE
===========

.. versionadded:: 0.16.0

Le traitement ``SENDMESSAGE`` envoie un message protocolaire à un partenaire
distant. Cela est notamment utilisé pour les acquittements applicatifs de bout
en bout dans les architectures *Store and Forward*.

Pour l'heure, cette tâche est **spécifique au protocole PeSIT**.

La tâche ouvre une connexion dédiée vers le partenaire, envoie le message configuré
ainsi que quelques métadonnées sur le transfert (id...) dépendantes du protocole,
puis ferme la connexion.

Les arguments de la tâche sont :

* **partner** (*string*) — Le nom du partenaire PeSIT distant vers lequel envoyer
  le message.
* **account** (*string*) — Le login du compte distant à utiliser pour
  l'authentification.
* **client** (*string*) - Le nom du client à utiliser pour envoyer le message.
* **message** (*string*, optionnel) — Le contenu textuel du message.

Exemple
-------

.. code-block:: yaml

   - type: SENDMESSAGE
     args:
       partner: "partenaire-emetteur"
       account: "mon-login"
       client: "client"
       message: "ACK transfer #TRANSFERID#"

