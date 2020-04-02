Lister les transferts
=====================

.. _RFC 3339: https://www.ietf.org/rfc/rfc3339.txt

.. http:get:: /api/transfers

   Renvoie une liste des transferts remplissant les critères donnés en
   paramètre de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les transferts seront triés *(défaut: start+)*
   :type sort: [start+|start-|id+|id-|status+|status-|rule_id+|rule_id-]
   :param remote: Filtre uniquement les transferts avec le partenaire renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs partenaires.
   :type remote: int
   :param account: Filtre uniquement les transferts avec le compte renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs comptes.
   :type account: int
   :param rule: Filtre uniquement les transferts avec la règle renseignée.
      Peut être renseigné plusieurs fois pour filtrer plusieurs règles.
   :type rule: int
   :param status: Filtre uniquement les transferts ayant le statut renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs status.
   :type status: [PLANNED|PRETASKS|TRANSFER|POSTTASKS|ERRORTASKS]
   :param start: Filtre uniquement les transferts dont la date est ultérieure à
      celle renseignée.
   :type start: date

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/transfers?limit=10&order=desc&rule=1&start=2019-01-01T01:00:00+02:00 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array transfers: La liste des transferts demandés
   :resjsonarr number id: L'identifiant unique du transfert
   :resjsonarr bool isServer: Précise si la gateway était à l'origine du transfert
   :resjsonarr number ruleID: L'identifiant de la règle de transfert
   :resjsonarr number agentID: L'identifiant du serveur de transfert
   :resjsonarr number accountID: L'identifiant du compte de transfert
   :resjsonarr string sourcePath: Le chemin d'origine du fichier
   :resjsonarr string destPath: Le chemin de destination du fichier
   :resjsonarr date start: La date de début du transfert
   :resjsonarr string status: Le statut actuel du transfert (*PLANNED*, *RUNNING*, *PAUSED* ou *INTERRUPTED*)
   :resjsonarr string step: L'étape actuelle du transfert (*PRE TASKS*, *DATA*, *POST TASKS* ou *ERROR TASKS*)
   :resjsonarr number progress: La progression (en octets) du transfert de données
   :resjsonarr number taskNumber: Le numéro du traitement en cours d'exécution
   :resjsonarr string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjsonarr string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 249

          {
            "transfers": [{
              "id": 1,
              "isServer": false,
              "ruleID": 1,
              "remoteID": 1,
              "accountID": 1,
              "source": "chemin/source/fichier1",
              "destination": "chemin/dest/fichier1",
              "start": "2019-01-01T02:00:00+02:00",
              "status": "RUNNING",
              "step": "DATA",
              "progress": 123456,
            },{
              "id": 2,
              "isServer": true,
              "ruleID": 1,
              "remoteID": 2,
              "accountID": 2,
              "source": "chemin/source/fichier2",
              "destination": "chemin/dest/fichier2",
              "start": "2019-01-01T03:00:00+02:00",
              "status": "PLANNED"
            }]
          }