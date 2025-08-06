Préenregistrer un transfert serveur
===================================

.. http:put:: /api/transfers

   Préenregistre un transfert serveur avec les informations fournies.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string rule: L'identifiant de la règle utilisée.
   :reqjson bool isSend: Indique si transfert est un envoi (``true``) ou une
     réception (``false``).
   :reqjson string server: Le nom du serveur local auquel le transfer est rattaché.
   :reqjson string account: Le nom du compte local qui fera la demande de transfert.
   :reqjson string file: Le nom du fichier à transférer.
   :reqjson date dueDate: La date d'expiration du transfert (en format ISO 8601).
     Une fois cette date passée, le transfert tombera en erreur.
   :reqjson object transferInfo: Des informations de transfert personnalisées sous
     la forme d'une liste de pairs clé:valeur, c'est-à-dire sous forme d'un objet JSON.

   :statuscode 201: Le transfert a été enregistré avec succès
   :statuscode 400: Un ou plusieurs des paramètres du transfert sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau transfert créé


   **Exemple de requête**

   .. code-block:: http

      PUT https://my_waarp_gateway.net/api/transfers HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 224

      {
        "rule": "règle_1",
        "isSend": true,
        "server": "serveur_sftp"
        "account": "toto",
        "file": "chemin/du/fichier",
        "dueDate": "2026-01-01T02:00:00+02:00",
        "transferInfo": { "key1": "val1", "key2": 2, "key3": true }
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/transfers/123
