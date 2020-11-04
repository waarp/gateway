Reprendre un transfert en pause
===============================

.. http:put:: /api/transfers/(int:transfer_id)/resume

   Reprend le transfert portant l'identifiant ``transfer_id`` si celui-ci est
   en pause, interrompu ou en erreur.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 201: Le transfert a été repris avec succès
   :statuscode 400: Le transfert demandé ne peut pas être repris
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resheader Location: Le chemin d'accès au transfert redémarré


   |

   **Exemple de requête**

      .. code-block:: http

         PUT https://my_waarp_gateway.net/api/transfers/1/resume HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/transfers/1