Consulter un transfert
======================

.. http:get:: /api/transfers/(int:transfer_id)

   Renvoie les informations du transfert portant l'identifiant ``transfer_id``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Les informations du transfert ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resjson number id: L'identifiant unique du transfert
   :resjson bool isServer: Précise si la gateway est à l'origine du transfert
   :resjson string rule: L'identifiant de la règle de transfert
   :resjson string requester: Le nom du compte ayant demandé le transfert
   :resjson string requested: Le nom du serveur/partenaire auquel le transfert a été demandé
   :resjson string sourcePath: Le chemin d'origine du fichier
   :resjson string destPath: Le chemin de destination du fichier
   :resjson date start: La date de début du transfert
   :resjson string status: Le statut actuel du transfert (*PLANNED*, *RUNNING*, *PAUSED* ou *INTERRUPTED*)
   :resjson string step: L'étape actuelle du transfert (*PRE TASKS*, *DATA*, *POST TASKS* ou *ERROR TASKS*)
   :resjson number progress: La progression (en octets) du transfert de données
   :resjson number taskNumber: Le numéro du traitement en cours d'exécution
   :resjson string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjson string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/transfers/1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 290

         {
           "id": 1,
           "isServer": true,
           "rule": "règle_1",
           "requester": "toto",
           "requested": "waarp_sftp",
           "sourcePath": "chemin/source/fichier1",
           "destPath": "chemin/dest/fichier1",
           "start": "2019-01-01T02:00:00+02:00",
           "status": "RUNNING",
           "step": "DATA",
           "progress": 123456,
         }