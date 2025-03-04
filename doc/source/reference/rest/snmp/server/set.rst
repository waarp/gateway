Modifier le serveur SNMP
========================

.. http:put:: /api/snmp/server

   Modifie la configuration du serveur SNMP. Si le serveur existe déjà, les champs
   omis resteront inchangés. Si le serveur SNMP n'existe pas, il sera créé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string localUDPAddress: L'adresse UDP locale du serveur (port inclus).
   :reqjson string community: [SNMPv2 uniquement] La valeur de communauté
      (ou mot de passe) du serveur. Par défaut, la valeur "public" est utilisée.
   :reqjson bool v3Only: Indique si le serveur ne doit accepter uniquement que
      les requêtes SNMPv3. Par défaut, le serveur accepte SNMPv2 et SNMPv3.
   :reqjson string v3Username: [SNMPv3 uniquement] Le nom d'utilisateur.
   :reqjson string v3AuthProtocol: [SNMPv3 uniquement] L'algorithme d'authentification
      utilisé. Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
      ``SHA-384`` et ``SHA-512``.
   :reqjson string v3AuthPassphrase: [SNMPv3 uniquement] La passphrase d'authentification.
   :reqjson string v3PrivProtocol: [SNMPv3 uniquement] L'algorithme de confidentialité
      utilisé. Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
      ``AES-256`` et ``AES-256C``.
   :reqjson string v3PrivPassphrase: [SNMPv3 uniquement] La passphrase de confidentialité.

   :statuscode 201: Le serveur a été créé/modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au serveur SNMP.


   **Exemple de requête**

   .. code-block:: http

      PUT https://my_waarp_gateway.net/api/snmp/server HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 242

      {
          "localUDPAddress": "0.0.0.0:161"
          "community": "public"
          "v3Only": false
          "v3Username": "waarp"
          "v3AuthProtocol": "SHA-256"
          "v3AuthPassphrase": "sesame"
          "v3PrivProtocol": "AES-256"
          "v3PrivPassphrase": "foobar"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/snmp/server
