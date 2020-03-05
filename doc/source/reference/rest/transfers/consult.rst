Consulter un transfert
======================

.. http:get:: /api/transfers/(int:transfer_id)

   Renvoie les informations du transfert portant l'identifiant ``transfer_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/transfers/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Les informations du transfert ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resjson number id: L'identifiant unique du transfert
   :resjson bool isServer: Précise si la gateway était à l'origine du transfert
   :resjson number ruleID: L'identifiant de la règle de transfert
   :resjson number agentID: L'identifiant du serveur de transfert
   :resjson number accountID: L'identifiant du compte de transfert
   :resjson string sourcePath: Le chemin d'origine du fichier
   :resjson string destPath: Le chemin de destination du fichier
   :resjson date start: La date de début du transfert
   :resjson string status: Le statut actuel du transfert (*PLANNED* ou *TRANSFER*)
   :resjson string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjson string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 107

          {
            "id": 1,
            "isServer": true,
            "ruleID": 1,
            "agentID": 1,
            "accountID": 1,
            "sourcePath": "chemin/source/fichier1",
            "destPath": "chemin/dest/fichier1",
            "start": "2019-01-01T02:00:00+02:00",
            "status": "TRANSFER"
          }