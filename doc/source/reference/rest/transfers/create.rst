Programmer un transfert
=======================

.. http:post:: /api/transfers

   Programme un nouveau transfert avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string ruleID: L'identifiant de la règle utilisée
   :reqjson string remoteID: L'identifiant du partenaire de transfert
   :reqjson number accountID: L'identifiant compte partenaire utilisé
   :reqjson string source: Le chemin du fichier source
   :reqjson string destination: Le chemin de destination du fichier

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/transfers HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 156

          {
            "ruleID": 1,
            "remoteID": 1,
            "accountID": 1,
            "source": "chemin/du/fichier",
            "destination": "chemin/de/destination"
          }

   **Réponse**

   :statuscode 202: Le transfert a été lancé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du transfert sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 202 ACCEPTED
