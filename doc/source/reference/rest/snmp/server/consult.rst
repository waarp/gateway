Consulter le serveur SNMP
=========================

.. http:get:: /api/snmp/server

   Renvoie la configuration du serveur SNMP.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le serveur a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La gateway n'a pas de serveur SNMP configuré

   :resjson string localUDPAddress: L'adresse UDP locale du serveur (port inclus).
   :resjson string community: [SNMPv2 uniquement] La valeur de communauté
      (ou mot de passe) du serveur.
   :resjson bool v3Only: Indique si le serveur ne doit accepter uniquement que
      les requêtes SNMPv3.
   :resjson string v3Username: [SNMPv3 uniquement] Le nom d'utilisateur.
   :resjson string v3AuthProtocol: [SNMPv3 uniquement] L'algorithme d'authentification
      utilisé. Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
      ``SHA-384`` et ``SHA-512``.
   :resjson string v3AuthPassphrase: [SNMPv3 uniquement] La passphrase d'authentification.
   :resjson string v3PrivProtocol: [SNMPv3 uniquement] L'algorithme de confidentialité
      utilisé. Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
      ``AES-256`` et ``AES-256C``.
   :resjson string v3PrivPassphrase: [SNMPv3 uniquement] La passphrase de confidentialité.

   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/snmp/server HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
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
