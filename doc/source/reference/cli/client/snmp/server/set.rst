=========================================
Modifier la configuration du serveur SNMP
=========================================

.. program:: waarp-gateway snmp server set

Modifie la configuration du serveur SNMP de la Gateway. Si la Gateway n'a pas de
serveur SNMP, celui-ci sera créé et démarré. Si le serveur est déjà actif, il
sera mis à jour puis redémarré.

**Commande**

.. code-block:: shell

   waarp-gateway snmp server set

**Options**

.. option:: -a <ADDRESS>, --udp-address=<ADDRESS>

   L'adresse UDP locale (port compris) du serveur SNMP.

.. option:: -c <COMMUNITY>, --community=<COMMUNITY>

   [SNMPv2 uniquement] La "communauté" SNMP du serveur. Dans les faits, cela
   correspond au mot de passe du serveur. Cette valeur est uniquement utilisée
   avec SNMPv2, car SNMPv3 gère l'authentification de manière différente (voir
   ci-dessous). Par défaut, la valeur "public" est utilisée.

.. option:: --v3-only

   Désactive le support de SNMPv2 sur le serveur. Celui-ci n'acceptera donc
   uniquement que les requêtes SNMPv3.

.. option:: --auth-username <AUTH_USERNAME>

   [SNMPv3 uniquement] Le nom d'utilisateur SNMPv3. Attention, omettre ce
   paramètre revient à désactiver l'authentification SNMPv3. Il est donc
   recommandé de toujours fournir une valeur ici, même si SNMPv3 n'est pas
   utilisé.

.. option:: --auth-protocol <AUTH_PROTOCOL>

   [SNMPv3 uniquement] L'algorithme d'authentification utilisé par le serveur.
   Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
   ``SHA-384`` et ``SHA-512``.

.. option:: --auth-passphrase <AUTH_PASSPHRASE>

   [SNMPv3 uniquement] La clé d'authentification SNMPv3.

.. option:: --priv-protocol <PRIVACY_PROTOCOL>

   [SNMPv3 uniquement] L'algorithme de chiffrage SNMPv3. Les valeurs acceptées
   sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``, ``AES-256`` et
   ``AES-256C``.

.. option:: --priv-passphrase <PRIVACY_PASSPHRASE>

   [SNMPv3 uniquement] La clé de chiffrage SNMPv3.

**Exemple**

.. code-block:: shell

   waarp-gateway snmp server set --udp-address "0.0.0.0:161" --community "public" --auth-username "waarp" --auth-protocol "SHA-512" --auth-passphrase "sesame" --priv-protocol "AES-256" --priv-passphrase "secret"
