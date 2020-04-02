Programmer un transfert
=======================

.. http:post:: /api/transfers

   Programme un nouveau transfert avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson bool isServer: Précise si la gateway était à l'origine du transfert
   :reqjson string ruleID: L'identifiant de la règle utilisée
   :reqjson string agentID: L'identifiant du serveur de transfert
   :reqjson number accountID: L'identifiant compte partenaire utilisé
   :reqjson string sourcePath: Le chemin du fichier source
   :reqjson string destPath: Le chemin de destination du fichier
   :reqjson date start: La date de début du transfert

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/transfers HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 156

          {
            "isServer": false,
            "ruleID": 1,
            "agentID": 1,
            "accountID": 1,
            "sourcePath": "chemin/du/fichier",
            "destPath": "chemin/de/destination",
            "start": "2019-01-01T02:00:00+02:00"
          }

   **Réponse**

   :statuscode 202: Le transfert a été lancé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du transfert sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau transfert créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 202 ACCEPTED
          Location: https://my_waarp_gateway.net/api/transfers/1
