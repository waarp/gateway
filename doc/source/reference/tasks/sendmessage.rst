.. _ref-task-sendmessage:

SENDMESSAGE
===========

.. versionadded:: 0.16.0

Le traitement ``SENDMESSAGE`` envoie un acquittement PeSIT (F.MESSAGE) à un
partenaire distant. Il est utilisé pour les acquittements applicatifs de bout
en bout dans les architectures *Store and Forward*.

Cette tâche est **spécifique au protocole PeSIT**. Si elle est configurée sur
une règle utilisée par un autre protocole, elle est silencieusement ignorée
(pas d'erreur de transfert).

Fonctionnement
--------------

La tâche ouvre une connexion PeSIT dédiée vers le partenaire, envoie un
F.MESSAGE contenant l'identifiant du transfert et un message texte, puis
ferme la connexion.

**Tous les paramètres sont optionnels** lorsque le mécanisme de routage
Store & Forward est configuré (voir ci-dessous). Le partenaire et le compte
sont alors résolus automatiquement depuis les infos de transfert.

Paramètres
----------

* **to** (*string*, optionnel) — Le nom du partenaire PeSIT distant vers
  lequel envoyer le message. Si omis, résolu depuis ``__replyPartner__``
  dans les infos de transfert (positionné automatiquement via PI 99
  ``REPLY=`` ou PI 61/PI 62). Supporte la substitution de variables.
* **as** (*string*, optionnel) — Le login du compte distant à utiliser pour
  l'authentification. Si omis, résolu depuis ``__replyAccount__`` dans les
  infos de transfert, ou le premier compte disponible sur le partenaire.
* **message** (*string*, optionnel) — Le contenu du F.MESSAGE. Si omis, un
  message ACK par défaut est généré (``ACK transfer <id>``). Supporte la
  substitution de variables (``#TRUEFILENAME#``, ``#TRANSFERID#``, etc.).
  Maximum 4096 caractères (PI 91).

Exemples
--------

**Acquittement automatique (zéro argument)** — le plus simple, recommandé
lorsque ``replyTo`` est configuré sur le partenaire émetteur :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE

**Acquittement avec message personnalisé** :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE
       args:
         message: "Fichier #TRUEFILENAME# recu avec succes"

**Acquittement explicite (partenaire et compte spécifiés)** :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE
       args:
         to: "partenaire-emetteur"
         as: "mon-login"
         message: "ACK transfer #TRANSFERID#"

**Acquittement conditionnel** (envoyé uniquement si le fichier est en EBCDIC) :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE
       condition: "#TI___fileEncoding__# == EBCDIC"

Résolution du partenaire
------------------------

Lorsque le paramètre ``to`` n'est pas renseigné, la tâche résout le
partenaire dans l'ordre suivant :

1. ``__replyPartner__`` dans les infos de transfert (positionné via PI 99
   ``REPLY=`` ou manuellement)
2. Premier partenaire PeSIT correspondant au ``__customerID__`` (PI 61)
3. Premier partenaire PeSIT correspondant au ``__bankID__`` (PI 62)

Lorsque le paramètre ``as`` n'est pas renseigné :

1. ``__replyAccount__`` dans les infos de transfert
2. Premier compte disponible sur le partenaire résolu

Intégration Store & Forward
----------------------------

Pour une chaîne A → B → C avec acquittement automatique :

1. **Partenaire** : configurer ``replyTo`` dans la :ref:`configuration
   protocolaire <proto-config-pesit>` de chaque partenaire
2. **Destinataire final (C)** : ajouter ``SENDMESSAGE`` en post-traitement
3. **Relais (B)** : activer ``relayMessages: true`` dans la configuration
   du serveur PeSIT

Voir la section :ref:`ref-proto-pesit` pour le détail du mécanisme de
routage et de relais automatique.
