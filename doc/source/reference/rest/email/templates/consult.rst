Consulter un template d'email
==============================

.. http:get:: /api/email/templates/(string:name)

   Renvoie le template d'email demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le template a été renvoyé avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: Le template demandé n'existe pas

   :resjson string name: Le nom du template.
   :resjson string subject: Le sujet de l'email.
   :resjson string mimeType: Le type MIME du corps de l'email.
   :resjson string body: Le corps de l'email.
   :resjson array attachments: La liste des fichiers joints au template.

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/email/templates/transfer-error HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 142

         {
           "name": "transfer-error",
           "subject": "Erreur de transfert",
           "mimeType": "text/plain",
           "body": "Le transfert {{ .Rule }} a échoué.",
           "attachments": []
         }
