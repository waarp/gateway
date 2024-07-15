=========================
Modifier un moniteur SNMP
=========================

.. program:: waarp-gateway snmp monitor update

Modifie un moniteur SNMP avec les paramètres donnés. Les paramètres omis
resteront inchangés.

**Commande**

.. code-block:: shell

   waarp-gateway snmp monitor update "<OLD_NAME>"

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nouveau nom du moniteur. Doit être unique.

.. option:: -a <UDP_ADDRESS>, --address=<UDP_ADDRESS>

   L'adresse UDP (avec le port) du moniteur SNMP à laquelle les notifications
   devront être envoyées.

.. option:: -v <VERSION>, --version=<VERSION>

   La version de SNMP à utiliser avec ce moniteur. Seules les valeurs ``SNMPv2``
   et ``SNMPv3`` sont acceptées, SNMPv1 n'est **pas** supporté.

.. option:: -t <NOTIFICATION_TYPE>, --notif-type=<NOTIFICATION_TYPE>

   Le type de notification à envoyer au moniteur. Les valeurs acceptées sont :
   ``trap`` ou ``inform``, la seule différence entre les deux étant que les
   *informs* doivent être acquittés par le moniteur, alors que les *traps* ne
   le sont pas. Par défaut, Waarp-Gateway utilisera des *traps*.

.. option:: -c <COMMUNITY>, --address=<UDP_ADDRESS>

   [SNMPv2 uniquement] La "communauté" SNMP du moniteur. Dans les faits, cela
   correspond au mot de passe du moniteur. Cette valeur est uniquement utilisée
   avec SNMPv2, car SNMPv3 gère l'authentification de manière différente (voir
   ci-dessous). Par défaut, la valeur "public" est utilisée.

.. option:: --context-name <CONTEXT_NAME>

   [SNMPv3 uniquement] *Optionnel* - Le contexte SNMPv3 des notifications.

.. option:: --context-engine-id <CONTEXT_ENGINE_ID>

   [SNMPv3 uniquement] *Optionnel* - L'ID du moteur de contexte SNMPv3. Inutile pour SNMPv2.

.. option:: --snmpv3-sec <SNMPV3_SECUTITY_LEVEL>

   [SNMPv3 uniquement] Le niveau de sécurité des notification SNMPv3. Ce paramètre
   permet d'activer ou désactiver l'authentification et le chiffrage des packets
   SNMP introduits avec la version 3 du protocole. Les valeurs acceptées sont :

      - ``noAuthNoPriv`` : pas d'authentification ni de chiffrage
      - ``authNoPriv`` : authentification, mais pas de chiffrage
      - ``authPriv`` : authentification ET chiffrage

   Par défaut, ni l'authentification, ni la confidentialité ne sont activées.

.. option:: --auth-engine-id <AUTH_ENGINE_ID>

   [SNMPv3 uniquement] L'identifiant d'authentification SNMPv3 de l'instance
   Waarp-Gateway. Ce paramètre est inutile dans le cas où le type de notification
   utilisé par le moniteur (option ``notif-type`` ci-dessus) est "*inform*".

.. option:: --auth-username <AUTH_USERNAME>

   [SNMPv3 uniquement] Le nom d'utilisateur SNMPv3. À noter que ce paramètre est
   requis dès lors que SNMPv3 est utilisé, et ce, même si l'authentification est
   désactivée.

.. option:: --auth-protocol <AUTH_PROTOCOL>

   [SNMPv3 uniquement] L'algorithme d'authentification utilisé par le moniteur.
   Ce paramètre est inutile si l'authentification SNMPv3 est désactivée
   via l'option ``snmpv3-sec``. Les valeurs acceptées sont : ``MD5``,
   ``SHA``, ``SHA-224``, ``SHA-256``, ``SHA-384`` et ``SHA-512``.

.. option:: --auth-passphrase <AUTH_PASSPHRASE>

   [SNMPv3 uniquement] La clé d'authentification SNMPv3. Inutile si
   l'authentification SNMPv3 est désactivée via l'option ``snmpv3-sec``.

.. option:: --priv-protocol <PRIVACY_PROTOCOL>

   [SNMPv3 uniquement] L'algorithme de chiffrage SNMPv3. Inutile si
   le chiffrage des notifications SNMPv3 est désactivé via l'option ``snmpv3-sec``.
   Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
   ``AES-256`` et ``AES-256C``.

.. option:: --priv-passphrase <PRIVACY_PASSPHRASE>

   [SNMPv3 uniquement] La clé de chiffrage SNMPv3. Inutile si le chiffrage
   des notifications SNMPv3 est désactivé via l'option ``snmpv3-sec``.

**Exemple**

.. code-block:: shell

   waarp-gateway snmp monitor update "nagios" -n "nagios-v3" -v "SNMPv3" --snmpv3-sec "authNoPriv" --auth-username "waarp" --auth-protocol "AES" --auth-passphrase "sesame"
